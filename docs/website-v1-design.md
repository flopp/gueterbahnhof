# Website v1 design decisions

## Product and audience

The site serves two related audiences: residents who need practical local
information and visitors who want to orient themselves and discover nearby
places. The primary tasks are to browse amenities, understand where a place is,
and read its essential details.

### Decision: German is the primary language

Write the interface and maintained place content in German. Set the document
language to `de`. A second language is not part of v1; keep wording and
templates structured so translations can be added later without duplicating
the generator.

### Decision: prioritize useful information over promotional content

The landing page should briefly explain the neighborhood and quickly link to
the directory and map. Place names, category, address, and practical details
should be more prominent than decorative copy.

**Reasoning:** A hyper-local information site is most useful when people can
complete a lookup quickly, especially on a phone.

## Information architecture

Use four page types:

1. **Home** — short introduction and prominent links to the directory and map.
2. **Directory** — all published places grouped or labeled by category; each
   item links to its detail page.
3. **Map** — a neighborhood map with selectable place links and a matching
   text list.
4. **Place detail** — a single place's description, address, opening hours,
   optional contact links, and optional images.

The global navigation should expose Home, Places, and Map consistently. Detail
pages should link back to the directory and map.

### Decision: use one canonical detail page per place

Each place has a stable slug and one canonical detail URL. Directory and map
links both point to that URL.

**Reasoning:** This avoids duplicating content and gives maintainers a single
place to update.

## Mobile-first visual direction

- Begin with a narrow, single-column layout; expand the directory and
  information layout at wider breakpoints.
- Keep text readable, use comfortable touch targets, and avoid horizontal
  scrolling.
- Use a restrained visual system: clear type hierarchy, consistent spacing,
  and a small set of colors tied to the neighborhood identity if approved
  assets or branding are available.
- Keep essential text and navigation visible without large hero imagery.
- Use cards or simple grouped lists for directory entries; each item should
  make its category and location easy to scan.
- On place pages, present address and opening hours before optional
  descriptive or photographic content.
- Treat the map as a visual aid, not the only navigation mechanism.

No exact colors, fonts, logo, or branded illustrations are specified yet.
Choose them after checking for existing project identity and licensed assets;
do not invent claims about the neighborhood's identity.

## Map behavior

### Decision: ship an interactive Leaflet map using OpenStreetMap data

The map page uses Leaflet with OpenStreetMap-based tiles and markers for places
with verified coordinates. Draw the neighborhood boundary from the supplied
[`data/gueterbahnhof.geojson`](../data/gueterbahnhof.geojson) polygon. Fit the
initial map view to that boundary, with enough surrounding context to orient
visitors. Selecting a marker opens a concise place summary with a link to its
detail page. The map also has a readable list of place links that works without
JavaScript and provides an accessible alternative to interacting with the map.

**Reasoning:** An interactive map makes location and nearby amenities easier
to explore while preserving static hosting. A parallel text list keeps place
discovery useful if JavaScript, tiles, or the map service are unavailable and
for visitors who do not use a visual map.

Do not request browser location permission in v1. Map tiles require a network
connection; when they cannot load, show a clear German message and retain the
place list and neighborhood boundary where possible. Display the required
OpenStreetMap and tile-provider attribution visibly on the map. Confirm the
selected tile provider's usage policy before release; do not assume the public
OpenStreetMap tile servers are an unrestricted production CDN.

## Content presentation

- Do not display empty labels or empty sections for optional data.
- Use a consistent German message such as “Keine Öffnungszeiten angegeben”
  only if it is useful and accurate; never imply that missing data means a
  place is closed.
- Keep descriptions concise and factual.
- Provide external phone, email, and website links only when verified.
- Show an information review date only when maintainers can keep it current.
- Provide meaningful alternative text for informative images; use empty alt
  text for purely decorative imagery.
- Include map/source attribution where its license requires it.

## Accessibility and resilience

- Use semantic headings, landmarks, labels, and link text.
- Ensure all functions are keyboard-operable and focus is visible.
- Preserve sufficient text/background contrast and do not encode category or
  status by color alone.
- Give the map an accessible title and provide equivalent place links in text.
- Keep core content and navigation available when JavaScript is disabled. The
  interactive map itself requires JavaScript; its place list remains available.
- Respect reduced-motion preferences if motion is introduced.

## Deferred design choices

- Search, category filters, and sorting should be added only when the first
  directory size and user feedback justify them.
- Language selection should wait until translations can be maintained as
  complete, accurate content.
- User submissions and personalized features require a later product and
  privacy design; they are outside v1.
