(() => {
  const element = document.getElementById("map");
  if (!element || !window.L) return;

  const status = document.getElementById("map-status");
  const fallback = element.querySelector(".map-fallback");
  if (fallback) fallback.remove();

  const map = L.map(element, { scrollWheelZoom: false, zoomControl: true });
  const tileUrl = element.dataset.tileUrl.replace(/%7B(z|x|y|s)%7D/gi, "{$1}");
  const tiles = L.tileLayer(tileUrl, {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>-Mitwirkende',
  }).addTo(map);
  map.setView([48.0144, 7.8473], 16);
  let tileLoaded = false;
  let tileFailed = false;
  const tileFallbackTimer = window.setTimeout(() => {
    if (!tileLoaded) {
      status.textContent = "Die Kartenkacheln konnten nicht geladen werden. Die Ortsliste bleibt verfügbar.";
      status.hidden = false;
    }
  }, 5000);
  tiles.on("tileerror", () => {
    tileFailed = true;
    status.textContent = "Die Kartenkacheln konnten nicht geladen werden. Die Ortsliste bleibt verfügbar.";
    status.hidden = false;
  });
  tiles.on("tileload", () => {
    tileLoaded = true;
    window.clearTimeout(tileFallbackTimer);
    if (!tileFailed) status.hidden = true;
  });

  const markerIcon = L.divIcon({
    className: "",
    html: '<span class="custom-marker"></span>',
    iconSize: [18, 18],
    iconAnchor: [9, 9],
  });
  const markerBounds = [];
  document.querySelectorAll(".map-place[data-lat][data-lon]").forEach((item) => {
    const lat = Number(item.dataset.lat);
    const lon = Number(item.dataset.lon);
    if (!Number.isFinite(lat) || !Number.isFinite(lon)) return;
    const point = [lat, lon];
    markerBounds.push(point);
    const name = item.querySelector("strong")?.textContent ?? item.textContent.trim();
    const category = item.querySelector("small")?.textContent ?? "";
    const popup = document.createElement("a");
    popup.className = "map-popup";
    popup.href = item.getAttribute("href");
    const title = document.createElement("strong");
    title.textContent = name;
    const subtitle = document.createElement("span");
    subtitle.textContent = category;
    popup.append(title, subtitle);
    L.marker(point, { icon: markerIcon }).addTo(map).bindPopup(popup);
  });

  fetch(element.dataset.boundaryUrl)
    .then((response) => {
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      return response.json();
    })
    .then((boundary) => {
      const layer = L.geoJSON(boundary, {
        style: {
          color: "#355446",
          weight: 3,
          opacity: 0.9,
          fillColor: "#99af8e",
          fillOpacity: 0.22,
        },
      }).addTo(map);
      const bounds = layer.getBounds();
      if (bounds.isValid()) {
        if (markerBounds.length) bounds.extend(markerBounds);
        map.fitBounds(bounds, { padding: [28, 28], maxZoom: 17 });
      }
    })
    .catch((error) => {
      console.error("Quartiersgrenze konnte nicht geladen werden:", error);
      status.textContent = "Die Quartiersgrenze konnte nicht geladen werden.";
      status.hidden = false;
    });

})();
