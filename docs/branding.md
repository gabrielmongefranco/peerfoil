<!--
This file is part of PeerFoil.
docs/branding.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Describes PeerFoil's artwork, placement, accessibility, and reuse, and carries the notice for the PNG exports under assets/brand/.
Notes: The SVG files under assets/brand/ are the editable masters.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil™ Brand Set

## The artwork, colors, and rules for showing PeerFoil consistently

[Return to the PeerFoil README](../README.md)

This page describes the PeerFoil artwork kept under `assets/brand/`: scalable SVG masters
and matching PNG exports for the banner, diagrams, logos, emblem, icons, and the GitHub
social preview. It explains where each file belongs, how the artwork stays accessible,
and the terms under which it may be reused.

## Files and placement

All artwork is in `assets/brand/`. SVGs are editable vector masters. PNGs are matching
raster exports. The SVG wordmark and emblem use custom paths, not an embedded font.
Taglines and diagram labels use system sans-serif fonts and can vary slightly by renderer.

| File stem | Size | Use |
| --- | --- | --- |
| `peerfoil-banner` | 1600 × 500 | Above the README H1 |
| `peerfoil-peer-review` | 900 × 310 | Under the opening section subheading |
| `peerfoil-workflow` | 900 × 810 | Under “How AI peer review works” |
| `peerfoil-workflow-mobile` | 480 × 880 | Narrow-screen diagram alternative |
| `peerfoil-social-preview` | 1280 × 640 | GitHub social preview; upload the PNG |
| `peerfoil-logo-dark` | 1100 × 280 | Transparent logo with dark lettering, for light surfaces |
| `peerfoil-logo-light` | 1100 × 280 | Transparent logo with light lettering, for dark surfaces |
| `peerfoil-mark` | 320 × 320 | Transparent mint and coral emblem |
| `peerfoil-mark-navy` | 320 × 320 | Single-color emblem for light surfaces |
| `peerfoil-mark-white` | 320 × 320 | Single-color emblem for dark surfaces |
| `peerfoil-icon-32`, `64`, `128`, `256`, `512` | Square | Transparent PNG icons |

## README placement

When the README shows the artwork, the banner goes above the H1, the peer-review
illustration under the opening subheading, and the workflow diagram under “How AI peer
review works”. The H1, the searchable text tagline, and the personal, credit, licensing,
and copyright sections stay as they are.

The banner uses `width="100%"`. Other images have a preferred display width and retain
their aspect ratio as their container narrows. The workflow uses a `picture` element to
select the narrow version below 600 pixels. Outside GitHub, apply
`img { max-width: 100%; height: auto; }` in your site's stylesheet.

## GitHub social preview

In the repository, open **Settings → General → Social preview → Edit → Upload an image**.
Choose `assets/brand/peerfoil-social-preview.png`. This file is 1280 × 640 and under 1 MB.
The SVG is its editable master, not the file to upload to this setting.

## Visual identity

| Color | Hex | Role |
| --- | --- | --- |
| Midnight navy | `#0B203F` | Background and dark logo |
| Mint | `#62E5A4` | P and first reviewer |
| Coral | `#FF7E6D` | Annotation and second reviewer |
| Warm white | `#FAFAF7` | Main text |
| Pale blue | `#BCD0E3` | Secondary text and connectors |

Keep the custom P's tapered stem, asymmetric bowl, and coral annotation together.
Keep the two review cards as a supporting detail, not a replacement for the main emblem.
Do not stretch, add cloud textures, or substitute the retired split-diamond symbol.
Use the supplied monochrome versions when color is unavailable. Allow clear space of at
least one stem-width around the emblem. At very small sizes, use the emblem without text.

## Accessibility and workflow meaning

Each SVG has a title and text description. Wherever the artwork appears, the page
supplies image alternatives and a nearby text equivalent of the workflow. Review roles
and transitions have written labels; their meaning does not rely on mint versus coral.
All artwork is static.

The diagram summarizes the planned workflow, not a claim that Core has shipped. The
architecture and plan are independently reviewed before production. Required checks
cannot be voted away. Accepted repairs return through validation and fresh independent
review. Only one automatic repair cycle is allowed; unresolved issues and review limits
require a user decision. An agent never approves its own work.

## Rights and limitations

Copyright © 2026 Gabriel Mongefranco. PeerFoil™ is a trademark of Gabriel Mongefranco.

These human-facing documentation assets, including corresponding PNG exports, use
GFDL-1.3-or-later with no Invariant Sections, Front-Cover Texts, or Back-Cover Texts,
consistent with the repository's documentation policy. This file also supplies the
copyright and license notice for PNG files, which cannot carry a comment.

Custom lettering and geometry do not establish trademark clearance or guaranteed
exclusivity. No completed reverse-image or trademark clearance is claimed for this set.

## Conclusion

The brand set gives PeerFoil one recognizable look across the README, social previews,
and icons while keeping the workflow meaning readable in text. Edit the SVG masters,
re-export the PNGs, and keep the notices in this file current when the artwork changes.

## Additional Resources

- [PeerFoil README](../README.md)
- [PeerFoil method](PeerFoil-Method.md)
- [GNU Free Documentation License, version 1.3](https://www.gnu.org/licenses/fdl-1.3.html)

[Return to the PeerFoil README](../README.md)

---

Copyright © 2026 Gabriel Mongefranco
