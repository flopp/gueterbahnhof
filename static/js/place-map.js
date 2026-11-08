(() => {
  const element = document.getElementById("place-map");
  if (!element || !window.L) return;

  const latitude = Number(element.dataset.lat);
  const longitude = Number(element.dataset.lon);
  if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) return;

  const status = document.getElementById("place-map-status");
  const fallback = element.querySelector(".map-fallback");
  if (fallback) fallback.remove();

  const map = L.map(element, { scrollWheelZoom: false, zoomControl: true });
  const tileUrl = element.dataset.tileUrl.replace(/%7B(z|x|y|s)%7D/gi, "{$1}");
  const tiles = L.tileLayer(tileUrl, {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>-Mitwirkende',
  }).addTo(map);
  const point = [latitude, longitude];
  map.setView(point, 17);
  const markerIcon = L.divIcon({
    className: "",
    html: '<span class="custom-marker"></span>',
    iconSize: [18, 18],
    iconAnchor: [9, 9],
  });
  L.marker(point, { icon: markerIcon }).addTo(map);

  let tileLoaded = false;
  let tileFailed = false;
  const tileFallbackTimer = window.setTimeout(() => {
    if (!tileLoaded) {
      status.textContent = "Die Kartenkacheln konnten nicht geladen werden.";
      status.hidden = false;
    }
  }, 5000);
  tiles.on("tileerror", () => {
    tileFailed = true;
    status.textContent = "Die Kartenkacheln konnten nicht geladen werden.";
    status.hidden = false;
  });
  tiles.on("tileload", () => {
    tileLoaded = true;
    window.clearTimeout(tileFallbackTimer);
    if (!tileFailed) status.hidden = true;
  });
})();
