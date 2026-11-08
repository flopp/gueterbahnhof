# Website v1 technical design

## Constraints and principles

- Generate a static site using Go and `html/template`, in line with
  [vision.md](./vision.md).
- Serve only generated files in production; the deployed host does not need
  Go.
- Keep content and presentation separate so routine place updates do not need
  generator changes.
- Prefer the Go standard library until a concrete need justifies a dependency.
- Fail the build on invalid content or missing referenced files instead of
  silently producing an incomplete site.
- Keep generation deterministic and avoid network access during a normal build.
- Generate German-language pages (`lang="de"`) for v1.

## Proposed repository layout

This is a target layout, not a description of files that already exist:

```text
cmd/generator/       CLI entry point
internal/site/       content loading, validation, and generation
content/places.json  source of truth for place entries
data/gueterbahnhof.geojson  neighborhood boundary supplied for the map
templates/           shared layout and page templates
static/               CSS, images, map, and other copied assets
static/vendor/leaflet/ pinned, self-hosted Leaflet distribution and license
.output/              generated site (build output; not source content)
```

The initial `content/places.json` may be an empty list until verified place
information is collected; the site must render clear German empty states rather
than invented listings. Add places by editing that file and placing referenced
images under `static/`.

The generator supports `-output`, `-content`, `-boundary`, `-templates`,
`-static`, `-base-path`, `-tile-url`, `-tile-attribution`,
`-tile-attribution-url`, and `-verbose`. Paths are resolved from the current
working directory. `-base-path` supports publication under a URL prefix, and
`-tile-url` selects an HTTPS OpenStreetMap-compatible tile URL template.
Attribution text and its HTTPS link can be configured for the selected tile
provider. The generator should return a non-zero exit status for invalid flags,
unreadable content, invalid data, missing assets, or write failures.

The Makefile provides `make` to build the generator, `make run` to generate
the site and serve it locally at `http://127.0.0.1:8000/` using Python's
built-in HTTP server, and `make test` to run `go test ./...`.

## Content model

Use one JSON file for the initial dataset. JSON can be read with the Go standard
library, is straightforward to review in pull requests, and does not require
introducing a parser dependency. Split it into multiple files only if real
maintenance needs arise.

Suggested shape:

```json
{
  "places": [
    {
      "slug": "example-place",
      "name": "Beispielort",
      "category": "Essen und Trinken",
      "description": "Eine kurze, sachliche Beschreibung.",
      "address": "Straße und Hausnummer, Freiburg",
      "openingHours": ["Montag-Freitag: ..."],
      "website": "https://example.invalid/",
      "phone": "+49 ...",
      "images": [
        {
          "src": "/images/example-place.jpg",
          "alt": "Außenansicht des Beispielorts"
        }
      ],
      "coordinates": {
        "latitude": 48.0123,
        "longitude": 7.8412
      },
      "source": "Quelle, anhand derer die Angaben geprüft wurden",
      "lastVerified": "2026-01-31"
    }
  ]
}
```

The sample values are illustrative only and must not be published as real
information.

### Validation rules

- Require `slug`, `name`, `category`, and `address` for a published place.
- Require slugs to be unique and to match a conservative URL-safe format such
  as lowercase letters, digits, and hyphens.
- Treat descriptions, hours, contact details, images, map coordinates, source,
  and verification date as optional.
- Validate optional URLs as absolute HTTP(S) URLs and phone numbers as plain
  text or `tel:` link targets after validation.
- Require image paths to be local paths under the static asset directory;
  reject traversal and missing files.
- If coordinates are present, validate latitude (`-90..90`) and longitude
  (`-180..180`) and document that they are WGS 84 decimal degrees, as expected
  by Leaflet.
- Parse verification dates in an agreed format (recommended: ISO `YYYY-MM-DD`).
- Present dates and any structured values using German conventions; preserve
  explicitly entered opening-hours text as maintained.
- Never use a placeholder example as published content.

Decide whether the initial release includes a `published` field or whether all
entries in the source file are publishable. Prefer a simple all-published
dataset until maintainers have a demonstrated draft workflow.

## Generation flow

1. Parse CLI flags and resolve input/template/static paths using documented
   defaults.
2. Read and decode the JSON dataset.
3. Read and validate `data/gueterbahnhof.geojson`. The supplied file is a
   GeoJSON `FeatureCollection` containing a `Polygon`; reject malformed JSON,
   unsupported geometry, unclosed rings, and out-of-range WGS 84 coordinates.
   Preserve the GeoJSON coordinate order (longitude, latitude).
4. Validate required place fields, uniqueness, URLs, coordinates, and asset
   references. Report the file and place slug for actionable errors.
5. Sort places deterministically (recommended: category, then name, with a
   stable slug tie-breaker).
6. Render shared HTML templates for the home, directory, map, and each detail
   page. Use `html/template` so content is contextually escaped.
7. Copy the GeoJSON boundary and static assets, including the Leaflet
   JavaScript/CSS assets and any locally hosted marker icons, into the output
   tree.
8. Return an explicit error on template execution, asset-copy, or filesystem
   failures; exit non-zero from the CLI.

Generate output paths such as:

```text
.output/index.html
.output/places/index.html
.output/places/<slug>/index.html
.output/map/index.html
.output/data/gueterbahnhof.geojson
.output/static/css/site.css
.output/static/images/...
```

Directory-style routes are convenient on static hosts that serve `index.html`
from a directory. Confirm host behavior and whether deployment uses a domain
root or subpath before finalizing URL generation. Keep URL construction
centralized so a later base-path requirement does not require template-wide
changes.

## Templates, assets, and browser behavior

- Use shared layout and partial templates to keep navigation and metadata
  consistent.
- Self-host the pinned Leaflet 1.9.4 release and its CSS/assets rather than
  depending on a third-party JavaScript CDN. Load map code on the map page and
  on place detail pages only when coordinates are available.
- Use an OpenStreetMap-based tile service for interactive tiles. Make its URL
  configurable without rebuilding templates, and follow that provider's
  current usage policy, caching requirements, and attribution rules. Do not
  treat `tile.openstreetmap.org` as an unrestricted production tile service.
- Keep a server-generated list of place links on the map page so place
  discovery remains available if JavaScript or tile requests fail.
- Load the local GeoJSON boundary into Leaflet with `L.geoJSON`, style it
  distinctly from place markers, and fit the initial map bounds to the
  boundary. Keep the polygon data local to the generated site; do not fetch it
  from the source repository at runtime.
- Initialize Leaflet only when the map container exists. Provide a visible
  German fallback message if tile loading fails; do not hide the place list.
- On a place detail page with coordinates, center a single marker on that
  place. Omit the detail map when coordinates are unavailable.
- Include visible map attribution identifying OpenStreetMap and the selected
  tile provider, with links where required by their policies.
- Provide useful page titles and descriptions; use canonical URLs only after a
  production domain is known.
- Keep all v1 interface strings in German and use German conventions for
  generated dates and number formatting.
- Use relative or correctly base-prefixed internal URLs appropriate to the
  confirmed hosting path.
- Use responsive image dimensions and suitable compression. Add lazy loading
  for below-the-fold content where appropriate.
- Use stable, descriptive filenames. The site remains static, but the
  interactive map specifically requires runtime JavaScript and network access
  to the tile provider.

## Testing and quality checks

- Unit-test JSON decoding, validation, slug uniqueness, URL checks, and
  coordinate boundaries.
- Test that malformed JSON, duplicate slugs, missing images, and invalid
  coordinates are reported as errors.
- Add an integration test that generates a small fixture dataset into a
  temporary directory, then checks expected output pages, escaped content,
  copied assets, and internal links.
- Verify deterministic output by generating the same fixture twice and
  comparing relevant files.
- Run `gofmt` and `go test ./...` for Go changes.
- Manually review the generated site at phone and desktop widths and with
  keyboard-only navigation. Check the Leaflet map at supported zoom levels,
  marker links, attribution, and behavior when tiles are unavailable.

## Deployment

Build the static files locally, then copy the generated output to the host with
`scp` as intended by the project vision. Keep credentials and host-specific
configuration outside the repository. Confirm the remote document root,
permissions, URL base path, and whether upload should replace or preserve
existing files before defining an automated deployment command.

Deployment must only copy generated static output. Do not expose source
datasets, generator binaries, or local development files unless intentionally
required.

## Later evolution

The JSON records and generated routes should remain easy to migrate into a
future backend, but v1 should not introduce database abstractions or an
administrative API prematurely. If an admin backend is later added, it should
reuse the validated place model where practical and preserve the static site as
a possible publishing target.

Keep visible interface strings and content in German for v1. If a second
language is later added, introduce explicit localized content and language
routes rather than relying on runtime machine translation.
