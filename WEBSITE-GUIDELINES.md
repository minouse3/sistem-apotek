# Website quality guidelines

Use [DESIGN.MD](DESIGN.MD) for the visual style and layout. Use these guidelines when writing interface copy, building features, or reviewing code.

## Interface copy

- Label actions by what they do: **Save changes**, **Add item**, **Download report**. Avoid vague labels like **Click here**.
- Remove descriptions that repeat the heading or explain an obvious control. Keep field labels, units, limits, and warnings people need to make a decision.
- Put errors next to the affected field or action. Say what went wrong and what the user can do: “Enter a whole number from 0 to 999,999.”
- Use the same terms for the same things across pages. Avoid buzzwords, filler, and unsupported claims.

## Code and comments

- Use clear names, semantic HTML, and existing Bootstrap components. Prefer direct logic over an abstraction with only one use and no clear benefit.
- Remove comments that repeat the code, narrate routine steps, or mark obvious sections. `// Open the dialog` adds nothing above `dialog.show()`.
- Keep short comments that explain a non-obvious business rule, browser workaround, or compatibility constraint. Preserve license notices and required documentation.
- If a block needs a comment just to tell readers what it does, first see whether a clearer name or simpler flow would help.

## What makes a site feel real

- Use believable sample records, clearly marked as demo data. Never present invented metrics, testimonials, or activity as real.
- Give every visible link and button a working destination or action. Omit controls for features that are not available. Show success only after the operation succeeds.
- Handle empty results, invalid input, and failed actions. Add a loading state only when there is something to wait for.
- Keep the main task easy to find. Use consistent spacing, readable contrast, visible keyboard focus, and text labels for status. Keep decorative elements out of the way.

Before shipping, open the page on desktop and mobile. Complete the main task with a keyboard, check the relevant empty and error states, and remove any copy or comments that add no information.
