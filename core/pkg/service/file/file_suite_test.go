// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package file_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/file"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestFile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service File Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db        *gorp.DB
	statusSvc *status.Service
	fileSvc   *file.Service
	tasks     *tasktestutil.Tasks
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	StartRun(db.Run)
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
	fileSvc = MustOpen(file.OpenService(ctx, file.ServiceConfig{DB: db}))
	tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
		DB:     db,
		Status: statusSvc,
		Stores: fileSvc.Stores(),
	}))
})

// listing and entry decode the data of a list reply.
type listing struct {
	Path       string  `json:"path"`
	Parent     string  `json:"parent"`
	SecretsDir string  `json:"secrets_dir"`
	Secret     bool    `json:"secret"`
	Truncated  bool    `json:"truncated"`
	Entries    []entry `json:"entries"`
}

type entry struct {
	Name        string               `json:"name"`
	Path        string               `json:"path"`
	Kind        string               `json:"kind"`
	Size        *int64               `json:"size"`
	Modified    *telem.TimeStamp     `json:"modified"`
	Certificate *certificate.Details `json:"certificate"`
}

// browser is a file browser task and its configured instance.
type browser struct {
	stored   task.Task
	instance driver.Task
}

// configure creates a file browser task and configures it with a factory that has the
// given secrets folder.
func configure(ctx context.Context, secretsDir string) browser {
	GinkgoHelper()
	f := MustSucceed(file.NewFactory(file.FactoryConfig{
		DB:         db,
		Status:     statusSvc,
		SecretsDir: secretsDir,
	}))
	key := MustSucceed(tasks.Create(ctx, fileSvc.Browser, msgpack.EncodedJSON{}))
	var t task.Task
	Expect(tasks.NewRetrieve().Where(task.MatchKeys(key)).Entry(&t).Exec(ctx, db)).
		To(Succeed())
	b := browser{
		stored:   t,
		instance: MustSucceed(f.ConfigureTask(ctx, t, driver.NoCommand)),
	}
	DeferCleanup(func() { Expect(b.instance.Stop(false)).To(Succeed()) })
	return b
}

// taskStatus retrieves the status of t.
func taskStatus(ctx context.Context, t task.Task) task.Status {
	GinkgoHelper()
	var stat task.Status
	Expect(statusSvc.NewRetrieve[task.StatusDetails]().
		Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
		Entry(&stat).
		Exec(ctx, db)).To(Succeed())
	return stat
}

// exec runs a list command with args on b and returns the status that answers it.
func (b browser) exec(ctx context.Context, args msgpack.EncodedJSON) task.Status {
	GinkgoHelper()
	return b.execType(ctx, "list", args)
}

// execType runs a command of type cmdType with args on b and returns the status that
// answers it.
func (b browser) execType(
	ctx context.Context,
	cmdType string,
	args msgpack.EncodedJSON,
) task.Status {
	GinkgoHelper()
	return b.send(ctx, ctx, cmdType, args)
}

// send runs a command on b with the time limit of cmdCtx, and returns the status that
// answers it.
func (b browser) send(
	ctx, cmdCtx context.Context,
	cmdType string,
	args msgpack.EncodedJSON,
) task.Status {
	GinkgoHelper()
	cmd := task.Command{
		Task: b.stored.Key,
		Type: cmdType,
		Key:  uuid.New().String(),
		Args: args,
	}
	Expect(b.instance.Exec(cmdCtx, cmd)).To(Succeed())
	var stat task.Status
	// A listing of a large folder can take over a second in a race build.
	Eventually(func(g Gomega) {
		stat = taskStatus(ctx, b.stored)
		g.Expect(stat.Details.Cmd).To(Equal(cmd.Key))
	}).WithTimeout(10 * time.Second).Should(Succeed())
	Expect(stat.Details.Running).To(BeTrue())
	return stat
}

// list lists path on b and returns the listing of a reply that succeeds.
func (b browser) list(ctx context.Context, path string) listing {
	GinkgoHelper()
	stat := b.exec(ctx, msgpack.EncodedJSON{"path": path})
	Expect(stat.Variant).To(Equal(status.VariantSuccess), stat.Description)
	Expect(stat.Message).To(Equal("Listed files"))
	var l listing
	Expect(stat.Details.Data.Unmarshal(&l)).To(Succeed())
	return l
}

// notAfter is when each certificate from newCertificate expires.
var notAfter = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

// newCertificate returns a self-signed certificate for the common name as DER, and the
// private key of the certificate as PEM.
func newCertificate(name string) (der, key []byte) {
	GinkgoHelper()
	priv := MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    notAfter.AddDate(-1, 0, 0),
		NotAfter:     notAfter,
	}
	der = MustSucceed(
		x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv),
	)
	return der, encodePEM("PRIVATE KEY", MustSucceed(x509.MarshalPKCS8PrivateKey(priv)))
}

func encodePEM(blockType string, der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
}
