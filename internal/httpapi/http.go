package httpapi

import (
	"net/http"
	"strings"

	"github.com/kosovrzn/wb-tech-l0/internal/cache"
	"github.com/kosovrzn/wb-tech-l0/internal/repo"
)

func NewHandler(store repo.Repository, c cache.Store) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!doctype html>
<html><head><meta charset="utf-8"><title>Order Lookup</title></head>
<body style="font-family:sans-serif;max-width:720px;margin:40px auto;">
<h3>Find Order</h3>
<input id="oid" placeholder="order_uid" style="width:420px;padding:6px;">
<button onclick="go()">Get</button>
<pre id="out" style="background:#f5f5f5;padding:12px;white-space:pre-wrap;word-break:break-word;font-family:ui-monospace,Menlo,Consolas,monospace;"></pre>
<script>
async function go(){
  const id = document.getElementById('oid').value.trim();
  if (!id) return;

  const out = document.getElementById('out');
  out.textContent = 'Loading...';

  const r = await fetch('/order/' + encodeURIComponent(id));
  if (!r.ok) { out.textContent = 'Not found'; return; }

  const data = await r.json(); // <-- парсим JSON
  out.textContent = JSON.stringify(data, null, 2); // <-- pretty print
}
</script>
</body></html>`))
	})

	mux.HandleFunc("/order/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/order/")
		if id == "" {
			http.Error(w, "missing order id", http.StatusBadRequest)
			return
		}

		if v, ok := c.Get(id); ok {
			w.Header().Set("X-Cache", "HIT")
			w.Header().Set("Content-Type", "application/json")
			w.Write(v)
			return
		}
		w.Header().Set("X-Cache", "MISS")

		raw, err := store.GetOrderRaw(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		c.Set(id, raw)
		w.Header().Set("Content-Type", "application/json")
		w.Write(raw)
	})

	return mux
}
