Hyper-local information website for Freiburg's new "Quartier Güterbahnhof" area.

Planned features:

- map view of the area
- directory of local shops, restaurants, other amenities
- details page for each amenity (description, opening hours, some pictures, etc.)

Considerations:
- It should be easy to add and modify amenity entries (first by modifying/adding files to the repositotry; later: by using an admin backend)
- The website should provide useful information to residents and visitors
- Nice visuals, mobile first usage

Technical:
- first:
  - static website - keep it simple
  - go based static site generator (go html templates)
  - deployable via scp
- later
  - go based backend (for submission, management, etc.)