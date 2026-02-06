# Phase 31: Modal UI Infrastructure - Context

**Gathered:** 2026-02-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Build a reusable modal dialog component for HTML reports that opens, displays content (traces, prompts), and closes correctly across desktop and mobile devices. Modal must be embedded in the HTML template and work with native browser APIs.

</domain>

<decisions>
## Implementation Decisions

### Modal sizing
- Desktop: Content-aware sizing (min 600px, max 90vw) - adapts to content while respecting viewport
- Mobile: Full-screen on mobile (no margins) - maximizes content space for reading traces/prompts

### Interaction patterns
- Closing methods: All three methods work (Esc key + X button + backdrop click)
- Initial focus: Close button receives focus when modal opens
- Content overflow: Claude's discretion (internal scroll vs entire modal scroll)
- Focus management: Claude's discretion (focus trap implementation)

### Claude's Discretion
- Visual style (animations, shadows, backdrop effect) - match existing HTML report design
- Backdrop appearance (dark overlay vs blur effect)
- Scroll behavior when content exceeds viewport
- Focus trap implementation details
- Exact CSS for responsive breakpoints

</decisions>

<specifics>
## Specific Ideas

- Use native `<dialog>` element with `showModal()` API for built-in modal behavior
- Full viewport utilization on mobile is important for reading long traces/prompts
- All three close methods (Esc, X button, backdrop) should work - maximizes user flexibility

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope

</deferred>

---

*Phase: 31-modal-ui-infrastructure*
*Context gathered: 2026-02-06*
