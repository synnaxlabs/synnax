 #!/usr/bin/env bash
  # Caps the memory of each CI runner on abc and defghi (#803). Opens SSH for this
  # machine's IP, and closes it on exit.
  set -euo pipefail
  SG=sg-02036935728d8727e
  HOSTS="i-02f671d6df2c88771 i-01b2e6836b90c9f65"
  CAP=10G

  ip=$(curl -s -m 5 https://checkip.amazonaws.com)
  dir=$(mktemp -d)
  ssh-keygen -q -t ed25519 -N '' -f "$dir/k"
  aws ec2 authorize-security-group-ingress --group-id $SG --protocol tcp --port 22 \
    --cidr "$ip/32" >/dev/null
  trap 'aws ec2 revoke-security-group-ingress --group-id $SG --protocol tcp --port 22 \
    --cidr "$ip/32" >/dev/null && echo "SSH rule for $ip removed"; rm -rf "$dir"' EXIT

  for id in $HOSTS; do
    read -r name addr <<<"$(aws ec2 describe-instances --instance-ids "$id" --query \
      'Reservations[0].Instances[0].[Tags[?Key==`Name`].Value|[0],PublicIpAddress]' \
      --output text)"
    echo "== $name ($addr)"
    aws ec2-instance-connect send-ssh-public-key --instance-id "$id" \
      --instance-os-user ubuntu --ssh-public-key "file://$dir/k.pub" >/dev/null
    ssh -i "$dir/k" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
      -o LogLevel=ERROR -o ConnectTimeout=15 "ubuntu@$addr" "CAP=$CAP bash -s" <<'EOF'
  set -eu
  units=$(systemctl list-units --plain --no-legend 'actions.runner.*' | awk '{print $1}')
  [ -n "$units" ] || { echo "no runner units"; exit 1; }
  for u in $units; do
    sudo mkdir -p "/etc/systemd/system/$u.d"
    printf '[Service]\nMemoryMax=%s\nOOMPolicy=continue\n' "$CAP" |
      sudo tee "/etc/systemd/system/$u.d/memory.conf" >/dev/null
  done
  sudo systemctl daemon-reload
  for u in $units; do
    max=/sys/fs/cgroup$(systemctl show -p ControlGroup --value "$u")/memory.max
    if [ "$(cat "$max")" = max ]; then
      # A restart kills a running job, so wait until this runner is idle.
      while systemd-cgls --no-pager -u "$u" | grep -q Runner.Worker; do
        echo "$u busy, waiting"; sleep 30
      done
      sudo systemctl restart "$u"
      max=/sys/fs/cgroup$(systemctl show -p ControlGroup --value "$u")/memory.max
    fi
    echo "$u memory.max=$(cat "$max") $(systemctl show -p OOMPolicy "$u")"
  done
  free -g | sed -n 2p
  EOF
  done
