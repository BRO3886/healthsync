// Dark mode toggle
(function () {
  var toggle = document.getElementById("dark-toggle");
  if (!toggle) return;

  toggle.addEventListener("click", function () {
    var html = document.documentElement;
    var current = html.getAttribute("data-theme");
    var next = current === "dark" ? "light" : "dark";
    html.setAttribute("data-theme", next);
    localStorage.setItem("theme", next);
  });
})();

// Mobile sidebar toggle
(function () {
  var hamburger = document.getElementById("nav-hamburger");
  var sidebar = document.getElementById("sidebar");
  if (!hamburger || !sidebar) return;

  hamburger.addEventListener("click", function () {
    sidebar.classList.toggle("open");
  });
})();
