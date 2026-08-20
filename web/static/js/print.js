// The printable sheet's two conveniences. Both are conveniences: the art picker
// is a real form with a real submit button, and the print dialog is one keyboard
// shortcut away, so the page works with this file blocked or absent.
(function () {
  const button = document.getElementById("print");
  if (button) {
    button.addEventListener("click", function () {
      window.print();
    });
  }

  // Changing the dropdown submits, so picking a treatment is one action rather
  // than two. The Apply button stays visible for anyone who never triggers a
  // change event.
  const select = document.getElementById("art");
  if (select && select.form) {
    select.addEventListener("change", function () {
      select.form.submit();
    });
  }

  // An A4 sheet is 794px wide on screen, so on a phone the preview would run off
  // the side of the page. Scale it to fit. Screen only, and a convenience like
  // the rest of this file: with the script blocked the sheets print at exactly
  // the same size, the preview just scrolls sideways.
  function fitSheets() {
    const scale = Math.min(1, (window.innerWidth - 32) / 794);
    document.documentElement.style.setProperty("--sheet-zoom", scale);
  }
  fitSheets();
  window.addEventListener("resize", fitSheets);
})();
