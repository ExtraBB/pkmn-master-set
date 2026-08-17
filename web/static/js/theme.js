// Dark-mode toggle. The theme itself is resolved before first paint by the inline
// script in templates/layout.html; this only handles the switch and the memory.
//
// Nothing is stored until the user actually picks a side, so an untouched browser
// keeps following the OS — including a change made while the page is open.
(function () {
  var STORE = "theme";
  var media = window.matchMedia("(prefers-color-scheme: dark)");

  function saved() {
    try { return localStorage.getItem(STORE); } catch (e) { return null; }
  }

  function apply(theme) {
    document.documentElement.dataset.theme = theme;
  }

  document.body.addEventListener("click", function (evt) {
    var button = evt.target.closest("[data-theme-toggle]");
    if (!button) return;
    var next = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
    apply(next);
    try { localStorage.setItem(STORE, next); } catch (e) {}
  });

  media.addEventListener("change", function (evt) {
    if (saved()) return;
    apply(evt.matches ? "dark" : "light");
  });
})();
