// Keyboard navigation for the search suggestions dropdown. The dropdown itself
// is fetched by htmx (see partials/search.html); this only adds arrow-key
// selection on top, auto-selecting the first result so Enter goes somewhere
// sensible even if the user never touched the arrow keys.
(function () {
  function options() {
    var list = document.getElementById("suggestions");
    return list ? Array.prototype.slice.call(list.querySelectorAll(".suggestion-list li")) : [];
  }

  function activate(items, next) {
    items.forEach(function (li) {
      li.setAttribute("aria-selected", "false");
      li.classList.remove("is-active");
    });
    if (next) {
      next.setAttribute("aria-selected", "true");
      next.classList.add("is-active");
      next.scrollIntoView({ block: "nearest" });
    }
  }

  document.body.addEventListener("htmx:afterSwap", function (evt) {
    if (evt.target.id !== "suggestions") return;
    var input = document.getElementById("q");
    var items = options();
    activate(items, items[0]);
    if (input) input.setAttribute("aria-expanded", items.length ? "true" : "false");
  });

  document.body.addEventListener("keydown", function (evt) {
    if (evt.target.id !== "q") return;
    var items = options();
    if (!items.length) return;

    if (evt.key === "ArrowDown" || evt.key === "ArrowUp") {
      evt.preventDefault();
      var current = items.findIndex(function (li) { return li.getAttribute("aria-selected") === "true"; });
      var next = evt.key === "ArrowDown"
        ? items[current < 0 ? 0 : (current + 1) % items.length]
        : items[current < 0 ? items.length - 1 : (current - 1 + items.length) % items.length];
      activate(items, next);
    } else if (evt.key === "Enter") {
      var active = items.find(function (li) { return li.getAttribute("aria-selected") === "true"; });
      var link = active && active.querySelector("a");
      if (link) {
        evt.preventDefault();
        window.location.href = link.getAttribute("href");
      }
    }
  });
})();
