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
})();
