# Accessibility status

The current build offers:

- GUI buttons with text labels and an explicit “selected” label for Yes/No choices; selection does not rely only on color.
- Default-button Enter operation, Gio keyboard-focus controls, Previous navigation, and scrollable content.
- Device-independent GUI layout and scalable text units. Multi-monitor/DPI acceptance testing remains pending.
- A keyboard-only terminal wizard with plain text, `B` for Previous, and no essential color-only information.
- Explicit CLI commands and JSON output for users who prefer their own terminal or assistive tools.
- Help opens the repository documentation in the default browser. The URL will move to Read the Docs later.

Screen-reader accessibility is **not yet verified**. We have not established Windows UI Automation exposure
or tested Narrator/NVDA. System high-contrast theme following, a complete focus-order/visible-focus audit,
zoom/text-scale acceptance, and accessible progress announcements remain work for the GUI sprint.
Do not describe this build as screen-reader accessible based solely on keyboard operation.

For now, the explicit CLI is the most predictable alternative to the graphical wizard; compatibility with
an individual screen reader depends on the user's terminal. The terminal wizard redraws its screen and is
not a substitute for testing assistive technology.
