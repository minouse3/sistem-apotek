# Sistem Apotek Sumber Waras design guide

These rules govern the shared shell and visual styling across pages. Each project defines its own routes, features, roles, navigation labels, and page content. A component does not have to appear on every page, but it must follow these rules wherever it appears.

## Required framework

Use [Bootstrap 5.3](https://getbootstrap.com/docs/5.3/getting-started/introduction/) as the UI foundation across all pages. Use its responsive grid, utilities, and components where they fit. Load the project's custom stylesheet after Bootstrap to apply this guide's colors, typography, and component styles. Include Bootstrap's JavaScript bundle when using interactive components such as offcanvas drawers or modals.

## Visual direction

A quiet, professional interface for everyday work. Use off-white backgrounds, white surfaces, and dark green accents. Let headings, numbers, and clear labels do the work. Keep decoration out of the way of the content.

## Page layout

| Viewport | Required shell |
| --- | --- |
| Desktop (`≥992px`) | Put branding and primary navigation in a `216px` white left sidebar. Use a `1px` right divider and keep the sidebar at the top of the viewport as the page scrolls. The main area sits beside it on the off-white canvas. Do not add a full-width desktop header above the sidebar and content. |
| Narrow (`<992px`) | Replace the sidebar with a Bootstrap offcanvas drawer. Show a `56px` minimum-height white top bar with a menu button and project name. Keep the same navigation and selected state in the drawer. |

Use this shell on pages that share primary navigation. Standalone pages without app-wide navigation can omit the sidebar and menu button.

The standalone sign-in page uses a centered white card on a sage backdrop. Inside the card, place the project name above the smaller sign-in heading, then equal-sized username and password fields. Keep the app's Bootstrap foundation, Roboto typography, forest action color, and visible focus. Put the password visibility icon inside the field without a separate border, and give the button an accessible name. Show a credential error above the fields and dismiss it when either credential changes.

When the sign-in form is submitted with empty fields, outline those fields in red and show a small Indonesian warning below each one instead of a browser popup. Clear each warning and highlight when the user edits that field.

- **Navigation:** Give each item at least `42px` of height and `7px` corners. Use muted text at rest, then sage fill and forest text for the current destination. Keep desktop branding in the sidebar; do not duplicate it above the content.
- **Main area:** Center the content within the available column. Use `width: min(100%, 1440px)` with padding included in the width. Set horizontal padding to `clamp(18px, 3vw, 48px)` and vertical padding to `38px`, reduced to `25px` below `576px`.
- **Page heading:** Start the main area with one page title on the canvas, not in a card or banner. Put optional supporting text beneath it. Align a page-level action to the right when there is room; wrap it below on narrow screens. Keep section actions beside their section title. If the app needs global controls such as a role switcher, place them in a compact utility group inside the main column on desktop, not in a full-width header.
- **Content:** Use `24px` after the heading and `16px` to `18px` between peer panels. Start with one primary column. Add a narrower secondary column only when it supports the main task; when used, allocate roughly `72%` and `28%` of the available width after the gap. Stack the secondary column below the primary below `1100px`.
- **Footer:** Omit by default. If required for links or notices, place it below the content within the main column and use muted text.

The page itself must not scroll horizontally. If a data table cannot reflow without losing information, confine horizontal scrolling to its table wrapper.

## Color

| Role | Color | Use |
| --- | --- | --- |
| Canvas | `#F7F8F5` | App background |
| Surface | `#FFFFFF` | Panels, dialogs, and form fields |
| Subtle surface | `#F8FAF7` | Table headers and inset notes |
| Text | `#202D25` | Body text and data |
| Muted text | `#5F6C62` | Supporting text, placeholders, and secondary controls |
| Divider | `#E7EBE5` | Panel outlines and row separators |
| Control border | `#D7DFD7` | Inputs and secondary buttons |
| Forest | `#274E3B` | Primary buttons, links, and keyboard focus |
| Deep forest | `#1D3D2D` | Display headings and primary-button hover or pressed states |
| Sage | `#EDF4EE` / `#274E3B` | Selected items and positive status labels |
| Amber | `#FBF3DF` / `#795419` | Warning labels |
| Rose | `#F9ECE9` / `#913C34` | Critical labels and inline errors |

Status pairs list the fill first and the text second. Always pair status color with a readable label.

## Typography

| Role | Style |
| --- | --- |
| Page title | `clamp(2rem, 3vw, 2.5rem)`, weight 650, `1.12` line height, `-0.035em` tracking, deep forest |
| Section heading | `19px`, weight 700, `1.25` line height, `-0.025em` tracking |
| Body | `15px`, weight 400, `1.5` line height |
| Controls and table text | `13px`; weight 600 for labels and 650 for buttons |
| Metadata | `11px` to `12px`, muted |
| Key figures | `30px`, weight 650, tabular numerals |

Load Roboto with weights 400 through 700 and set Bootstrap’s `--bs-body-font-family` to `Roboto, system-ui, sans-serif`. Use that stack for headings, controls, and tables. Write labels in sentence case; reserve uppercase for small group labels.

## Component finish

- **Panels and dialogs:** White fill, `10px` corners, and no decorative shadows. Use a `1px` divider-colored border on panels and `16px` to `20px` padding for standard panel content. Separate list rows with rules instead of wrapping each row in its own card.
- **Buttons:** `7px` corners and `36px` minimum height for standard buttons. Use white text on forest for primary actions, deep forest on hover or press, and a transparent fill with a control border for secondary actions.
- **Inputs and selects:** White fill, control border, `7px` corners, and `37px` minimum height. Use muted placeholder text and keep visible labels on form fields.
- **Tables:** Use a subtle-surface header, white rows, and `1px` row dividers. Use about `8px` of vertical cell padding, `#FAFBF9` for row hover, and tabular numerals for quantities. Do not add zebra stripes.
- **Status badges:** Use semantic fill and text pairs, `11px` text at weight 650, `3px 7px` padding, and a `5px` dot in the text color. Reserve pill shapes for status labels.
- **Icons:** Use Phosphor Bold at `15px` to `17px` for standard controls and navigation. Do not mix icon families. Keep `6px` to `10px` between closely related elements.

## Interaction and copy

Use a visible 2 px forest keyboard outline with a 3 px offset. For focused inputs, use a `#72917D` border with a soft `#DCEBE0` ring. Keep feedback brief; no looping effects, gradients, glass effects, or heavy shadows.

Use direct action labels and plain status text. Explain limits and errors only where the reader needs them.

## Review before shipping

- At `1365px`, check for the white `216px` sidebar, no full-width header, and a `40px` page title at the default root font size.
- At `390px`, check for the compact top bar, working offcanvas navigation, a `32px` page title at the default root font size, and no page-level horizontal scroll.
- Check that Roboto loads and that buttons, tables, and badges use the shared rules on more than one page. Fix differences in the shared stylesheet rather than overriding them page by page.
