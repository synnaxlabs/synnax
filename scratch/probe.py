import sys, gi
gi.require_version("Gtk", "3.0")
gi.require_version("WebKit2", "4.1")
from gi.repository import Gtk, WebKit2, GLib

HTML = """<html><body><canvas id=c></canvas><script>
const out = {};
try { out.main = !!document.getElementById('c').getContext('webgl2'); } catch (e) { out.main = 'err ' + e; }
try { out.offscreenMain = !!new OffscreenCanvas(10,10).getContext('webgl2'); } catch (e) { out.offscreenMain = 'err ' + e; }
try {
  const gl = document.createElement('canvas').getContext('webgl2');
  const dbg = gl && gl.getExtension('WEBGL_debug_renderer_info');
  out.renderer = dbg ? gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) : (gl ? gl.getParameter(gl.RENDERER) : null);
} catch (e) { out.renderer = 'err ' + e; }
const src = "onmessage=()=>{let r;try{r=!!new OffscreenCanvas(10,10).getContext('webgl2')}catch(e){r='err '+e}postMessage(r)}";
try {
  const w = new Worker(URL.createObjectURL(new Blob([src])));
  w.onmessage = (e) => { out.offscreenWorker = e.data; document.title = JSON.stringify(out); };
  w.onerror = (e) => { out.offscreenWorker = 'worker error ' + e.message; document.title = JSON.stringify(out); };
  w.postMessage(1);
} catch (e) { out.offscreenWorker = 'err ' + e; document.title = JSON.stringify(out); }
setTimeout(() => { document.title = JSON.stringify(out); }, 5000);
</script></body></html>"""

win = Gtk.Window()
view = WebKit2.WebView()
settings = view.get_settings()
settings.set_enable_webgl(True)
if len(sys.argv) > 1:
    feats = WebKit2.Settings.get_all_features()
    for i in range(feats.get_length()):
        f = feats.get(i)
        ident = f.get_identifier()
        if any(k in ident.lower() for k in ("offscreen", "webgl", "canvas", "worker")):
            print("feature", ident, f.get_status(), settings.get_feature_enabled(f), flush=True)
            if ident == "AllowWebGLInWorkers":
                settings.set_feature_enabled(f, True)
win.add(view)
win.set_default_size(400, 300)
win.show_all()
def on_title(v, _):
    t = v.get_title()
    if t and t.startswith("{"):
        print("WebKitGTK", WebKit2.get_major_version(), WebKit2.get_minor_version(), WebKit2.get_micro_version(), t, flush=True)
        Gtk.main_quit()
view.connect("notify::title", on_title)
view.load_html(HTML, "about:blank")
GLib.timeout_add_seconds(20, Gtk.main_quit)
Gtk.main()
