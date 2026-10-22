// Search and tag filters for the catalogue (WEB-4), copy buttons (WEB-5).
(function () {
  var cards = Array.prototype.slice.call(document.querySelectorAll("#cards .card"));
  var q = document.getElementById("q");
  var chips = Array.prototype.slice.call(document.querySelectorAll(".chip"));
  var none = document.getElementById("none");
  if (cards.length && q) {
    var params = new URLSearchParams(location.search);
    var tag = params.get("tag") || "";
    q.value = params.get("q") || "";
    var apply = function () {
      var words = q.value.toLowerCase().split(/\s+/).filter(Boolean);
      var shown = 0;
      cards.forEach(function (c) {
        var ok = words.every(function (w) { return c.dataset.search.indexOf(w) >= 0; }) &&
          (!tag || (" " + c.dataset.tags + " ").indexOf(" " + tag + " ") >= 0);
        c.hidden = !ok;
        if (ok) shown++;
      });
      chips.forEach(function (ch) { ch.setAttribute("aria-pressed", String(ch.dataset.tag === tag)); });
      if (none) none.hidden = shown > 0;
      var p = new URLSearchParams();
      if (q.value) p.set("q", q.value);
      if (tag) p.set("tag", tag);
      var s = p.toString();
      history.replaceState(null, "", location.pathname + (s ? "?" + s : "") + location.hash);
    };
    q.addEventListener("input", apply);
    q.form.addEventListener("submit", function (e) { e.preventDefault(); apply(); });
    chips.forEach(function (ch) { ch.addEventListener("click", function () { tag = ch.dataset.tag; apply(); }); });
    apply();
  }
  document.querySelectorAll(".copy").forEach(function (b) {
    b.addEventListener("click", function () {
      if (!navigator.clipboard) return;
      navigator.clipboard.writeText(b.dataset.copy).then(function () {
        var t = b.textContent; b.textContent = "Copied"; setTimeout(function () { b.textContent = t; }, 1500);
      });
    });
  });
})();
