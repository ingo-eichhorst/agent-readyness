# Phase 31: Modal UI Infrastructure - Research

**Researched:** 2026-02-06
**Domain:** HTML `<dialog>` element, CSS modal patterns, accessibility, iOS Safari scroll locking
**Confidence:** HIGH

## Summary

This phase adds a reusable modal dialog component to the existing HTML report template. The native HTML `<dialog>` element with `showModal()` is the correct approach -- it provides built-in modal behavior including backdrop, Escape key handling, focus trapping, and `aria-modal` without any JavaScript libraries. The `<dialog>` element has been baseline-available across all major browsers since March 2022.

The main implementation challenges are: (1) closing on backdrop click requires a workaround since `<dialog>` does not natively support this (the newer `closedby="any"` attribute lacks Safari support as of early 2026), (2) iOS Safari requires a `position:fixed` body workaround to prevent background scrolling, and (3) the modal CSS and JS must be inlined into the existing Go `embed.FS` template system (no external dependencies).

**Primary recommendation:** Use native `<dialog>` with `showModal()`, implement backdrop-click-to-close via a click target check on the dialog element itself, and add the iOS Safari scroll lock using `position:fixed` on body with scroll position save/restore.

## Standard Stack

This phase uses zero external libraries. Everything is native browser APIs.

### Core
| Technology | Version | Purpose | Why Standard |
|------------|---------|---------|--------------|
| HTML `<dialog>` | Baseline since 2022 | Modal container element | Built-in modal behavior, focus trap, backdrop, Escape key |
| `showModal()` | Web API | Opens dialog as modal | Provides inertness, backdrop, focus management for free |
| `::backdrop` | CSS pseudo-element | Overlay behind modal | Native to `<dialog>`, no extra DOM needed |

### Supporting
| Technology | Purpose | When to Use |
|------------|---------|-------------|
| CSS custom properties | Theming consistency | Reuse existing `--color-*` variables from styles.css |
| `autofocus` attribute | Initial focus control | Place on close button to meet accessibility requirement |
| Go `embed.FS` | Template embedding | Existing pattern -- CSS/JS inlined into HTML |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `<dialog>` | Custom div + overlay | Loses built-in focus trap, Escape, aria-modal, inertness |
| `closedby="any"` | Manual backdrop click handler | `closedby` lacks Safari support; manual approach works everywhere |
| CSS `overscroll-behavior: contain` | `position:fixed` body hack | `overscroll-behavior` does not fully prevent body scroll on iOS Safari |

**Installation:** None -- all native browser APIs.

## Architecture Patterns

### Where Modal Code Lives

The modal is entirely within the existing template files:

```
internal/output/templates/
  report.html    -- Add <dialog> element + JS for open/close
  styles.css     -- Add modal CSS (dialog, backdrop, responsive)
```

No new Go files are needed. The `html.go` already embeds these templates via `//go:embed`.

### Pattern 1: Native Dialog Modal

**What:** A `<dialog>` element in the HTML template that is opened via `showModal()` and populated with content dynamically.

**When to use:** Every modal in the report (evidence detail, traces, prompts).

**Structure in report.html:**
```html
<!-- Single reusable modal at bottom of body, before </body> -->
<dialog id="ars-modal" class="ars-modal">
  <div class="ars-modal-content">
    <header class="ars-modal-header">
      <h3 class="ars-modal-title"></h3>
      <button class="ars-modal-close" autofocus aria-label="Close">&times;</button>
    </header>
    <div class="ars-modal-body"></div>
  </div>
</dialog>
```

Key design decisions:
- Single `<dialog>` element reused for all modals (content swapped via JS)
- Inner `.ars-modal-content` div is needed for backdrop click detection
- `autofocus` on close button ensures predictable initial focus
- `aria-label="Close"` on the X button for screen readers

### Pattern 2: Backdrop Click Detection

**What:** Detecting clicks on the backdrop (outside modal content) to close the dialog.

**Why needed:** `<dialog>` does not natively close on backdrop click. The `closedby="any"` attribute is not supported in Safari as of early 2026.

**Implementation:**
```javascript
// The dialog element itself is the click target when backdrop is clicked.
// Child elements are the target when clicking inside content.
dialog.addEventListener('click', (e) => {
  if (e.target === dialog) {
    dialog.close();
  }
});
```

This works because the `::backdrop` pseudo-element is part of the dialog's rendering, and clicks on it register with `e.target === dialog`. Clicks on content inside the dialog have child elements as targets.

**Critical CSS requirement:** The inner content div must not fill the entire dialog box -- there must be padding/gap so that clicks on the dialog (outside content) are possible:

```css
.ars-modal {
  padding: 0;  /* Remove default dialog padding */
}

.ars-modal-content {
  /* Content does not fill entire dialog click area */
  margin: auto;  /* Centers in the dialog */
}
```

### Pattern 3: iOS Safari Scroll Lock

**What:** Preventing background page scroll when modal is open on iOS Safari.

**Why needed:** iOS Safari ignores `overflow: hidden` on body for touch scrolling. The `overscroll-behavior: contain` CSS property does not fully solve this on iOS Safari.

**Implementation:**
```javascript
function openModal(title, bodyHTML) {
  const dialog = document.getElementById('ars-modal');
  dialog.querySelector('.ars-modal-title').textContent = title;
  dialog.querySelector('.ars-modal-body').innerHTML = bodyHTML;

  // iOS Safari scroll lock
  document.body.dataset.scrollY = window.scrollY;
  document.body.style.position = 'fixed';
  document.body.style.top = `-${window.scrollY}px`;
  document.body.style.left = '0';
  document.body.style.right = '0';

  dialog.showModal();
}

function closeModal() {
  const dialog = document.getElementById('ars-modal');
  dialog.close();

  // Restore scroll position
  const scrollY = document.body.dataset.scrollY || '0';
  document.body.style.position = '';
  document.body.style.top = '';
  document.body.style.left = '';
  document.body.style.right = '';
  window.scrollTo(0, parseInt(scrollY));
}
```

### Anti-Patterns to Avoid

- **Using `dialog.show()` instead of `dialog.showModal()`:** `.show()` does not create a backdrop, does not trap focus, and does not make the page inert. Always use `showModal()`.
- **Adding `role="dialog"` or `aria-modal="true"` manually:** The `<dialog>` element with `showModal()` sets these automatically. Adding them manually can cause duplicate announcements.
- **Nesting interactive elements outside the dialog while it is modal:** Everything outside the dialog is inert when `showModal()` is active. Popovers or tooltips must be inside the dialog.
- **Setting `tabindex` on the `<dialog>` element itself:** The dialog is not an interactive element. Use `autofocus` on a child element instead.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Modal overlay/backdrop | Custom div overlay + z-index stacking | `<dialog>` + `::backdrop` | Native backdrop handles stacking context, pointer events correctly |
| Focus trapping | Manual focus trap with keydown listener | `showModal()` built-in inertness | Browser handles focus trap, tab cycling, and restoring focus on close |
| Escape key handling | keydown event listener for Escape | `showModal()` built-in Escape | Browser fires `close` event automatically on Escape press |
| Page inertness | `inert` attribute on body children | `showModal()` auto-inertness | Browser makes everything outside dialog inert automatically |
| Scroll position save/restore | Custom scroll manager | Simple `window.scrollY` + `body.style.top` pattern | Well-established 4-line pattern, no abstraction needed |

**Key insight:** The `<dialog>` element with `showModal()` provides 80% of modal functionality for free (backdrop, focus trap, Escape, inertness, ARIA). The only manual work is backdrop-click-to-close and iOS scroll lock.

## Common Pitfalls

### Pitfall 1: Dialog Content Fills Entire Click Area
**What goes wrong:** Backdrop click detection fails because clicks always land on child elements, never on the dialog itself.
**Why it happens:** Default `<dialog>` padding or content that stretches to fill the dialog.
**How to avoid:** Set `padding: 0` on dialog, use an inner `.ars-modal-content` wrapper that does not stretch to fill the entire dialog. The dialog itself acts as the backdrop click target.
**Warning signs:** Clicking outside content area does not close the modal.

### Pitfall 2: Modal Opens Scrolled to Bottom
**What goes wrong:** If `autofocus` is on an element at the bottom of the dialog, or the first focusable element is at the bottom, the modal opens scrolled down.
**Why it happens:** `showModal()` scrolls to bring the focused element into view.
**How to avoid:** Put `autofocus` on the close button in the header (top of dialog). This is also the accessibility best practice per the decisions.
**Warning signs:** Modal content appears scrolled on open.

### Pitfall 3: iOS Body Scroll Not Fully Locked
**What goes wrong:** On iOS Safari, background page scrolls when user swipes inside the modal.
**Why it happens:** iOS Safari does not respect `overflow: hidden` on body for touch events. CSS `overscroll-behavior: contain` is insufficient.
**How to avoid:** Use the `position: fixed` + scroll position save/restore pattern on body element. Set `left: 0; right: 0;` in addition to `top` to prevent horizontal shift.
**Warning signs:** Background content moves when scrolling modal on iPhone.

### Pitfall 4: Scroll Position Lost After Close
**What goes wrong:** Page jumps to top when modal closes.
**Why it happens:** Setting `position: fixed` on body resets scroll. Forgetting to save/restore `window.scrollY`.
**How to avoid:** Save `window.scrollY` before setting `position: fixed`, restore with `window.scrollTo()` after removing it.
**Warning signs:** Page position changes after closing modal.

### Pitfall 5: Scrollbar Layout Shift
**What goes wrong:** Page content shifts horizontally when modal opens/closes because the scrollbar disappears/reappears.
**Why it happens:** `position: fixed` removes the scrollbar, changing available width.
**How to avoid:** Use `scrollbar-gutter: stable` on the root element, or add `padding-right` equal to scrollbar width when locking. The `scrollbar-gutter` approach is cleaner:
```css
html {
  scrollbar-gutter: stable;
}
```
**Warning signs:** Content jumps left/right when modal opens/closes.

### Pitfall 6: closedby Attribute Not Cross-Browser
**What goes wrong:** Relying on `closedby="any"` for backdrop click closure fails in Safari.
**Why it happens:** Safari has not shipped `closedby` support as of early 2026 (blocking baseline status since July 2025).
**How to avoid:** Do not use `closedby`. Implement backdrop click manually via event target check.
**Warning signs:** Backdrop click works in Chrome but not Safari.

## Code Examples

### Complete Modal CSS

```css
/* Modal dialog */
.ars-modal {
  border: none;
  border-radius: 0.5rem;
  padding: 0;
  max-width: min(90vw, 700px);
  max-height: 85vh;
  width: 100%;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  overflow: visible;
}

.ars-modal::backdrop {
  background: rgba(0, 0, 0, 0.5);
}

.ars-modal-content {
  display: flex;
  flex-direction: column;
  max-height: 85vh;
}

.ars-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.ars-modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.ars-modal-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: var(--color-muted);
  padding: 0.25rem 0.5rem;
  line-height: 1;
  border-radius: 0.25rem;
}

.ars-modal-close:hover {
  background: var(--color-border);
  color: var(--color-text);
}

.ars-modal-body {
  padding: 1.5rem;
  overflow-y: auto;
  flex: 1;
  min-height: 0;  /* Needed for flexbox scroll */
}

/* Mobile: full-screen modal */
@media (max-width: 640px) {
  .ars-modal {
    max-width: 100vw;
    max-height: 100vh;
    height: 100vh;
    width: 100vw;
    border-radius: 0;
    margin: 0;
  }

  .ars-modal-content {
    max-height: 100vh;
    height: 100%;
  }
}
```

### Complete Modal JavaScript

```javascript
// Modal open/close functions
function openModal(title, bodyHTML) {
  var dialog = document.getElementById('ars-modal');
  dialog.querySelector('.ars-modal-title').textContent = title;
  dialog.querySelector('.ars-modal-body').innerHTML = bodyHTML;

  // iOS Safari scroll lock
  document.body.dataset.scrollY = window.scrollY;
  document.body.style.position = 'fixed';
  document.body.style.top = '-' + window.scrollY + 'px';
  document.body.style.left = '0';
  document.body.style.right = '0';

  dialog.showModal();
}

function closeModal() {
  var dialog = document.getElementById('ars-modal');
  dialog.close();
}

// Setup event listeners
(function() {
  var dialog = document.getElementById('ars-modal');
  if (!dialog) return;

  // Close button
  dialog.querySelector('.ars-modal-close').addEventListener('click', closeModal);

  // Backdrop click (target is dialog itself, not child elements)
  dialog.addEventListener('click', function(e) {
    if (e.target === dialog) {
      closeModal();
    }
  });

  // Restore scroll on close (handles Escape key, X button, backdrop)
  dialog.addEventListener('close', function() {
    var scrollY = document.body.dataset.scrollY || '0';
    document.body.style.position = '';
    document.body.style.top = '';
    document.body.style.left = '';
    document.body.style.right = '';
    window.scrollTo(0, parseInt(scrollY));
  });
})();
```

### Dialog HTML Template Element

```html
<dialog id="ars-modal" class="ars-modal">
  <div class="ars-modal-content">
    <header class="ars-modal-header">
      <h3 class="ars-modal-title"></h3>
      <button class="ars-modal-close" autofocus aria-label="Close">&times;</button>
    </header>
    <div class="ars-modal-body"></div>
  </div>
</dialog>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Custom div overlay + JS focus trap | Native `<dialog>` + `showModal()` | Baseline March 2022 | Eliminates hundreds of lines of JS; built-in a11y |
| `overflow: hidden` on body | `position: fixed` + scroll save | Ongoing (iOS-specific) | Required for iOS Safari scroll lock |
| Manual `aria-modal` + `role="dialog"` | Implicit from `<dialog>` element | Baseline March 2022 | Less boilerplate, fewer a11y mistakes |
| Manual backdrop click via `closedby="any"` | Event target check on dialog | `closedby` not yet cross-browser | Manual approach needed until Safari ships `closedby` |

**Deprecated/outdated:**
- Custom focus trap libraries (focus-trap, a11y-dialog): Unnecessary when using `showModal()` which provides native focus trapping
- `closedby` attribute: Not yet usable due to Safari gap; use manual backdrop click handler instead

## Open Questions

1. **scrollbar-gutter browser support**
   - What we know: `scrollbar-gutter: stable` prevents layout shift when scrollbar appears/disappears. It is supported in Chrome, Firefox, and Safari 17.4+.
   - What's unclear: Whether it works correctly in combination with `position: fixed` body scroll lock.
   - Recommendation: Add it as a progressive enhancement. If it causes issues, fall back to `padding-right` compensation.

2. **autofocus on close button vs heading**
   - What we know: Decision says close button should receive focus. MDN recommends `autofocus` on the element the user should interact with first.
   - What's unclear: Whether screen readers will announce the modal title before focus lands on close button.
   - Recommendation: Place heading before close button in DOM order. Screen readers typically announce dialog content in DOM order regardless of focus target. Use `autofocus` on close button as decided.

## Sources

### Primary (HIGH confidence)
- [MDN `<dialog>` element](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/dialog) - showModal() API, close() API, backdrop, focus, browser support, accessibility
- [MDN `::backdrop`](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Selectors/::backdrop) - Backdrop pseudo-element styling

### Secondary (MEDIUM confidence)
- [CSS-Tricks: Prevent Page Scrolling When a Modal is Open](https://css-tricks.com/prevent-page-scrolling-when-a-modal-is-open/) - iOS Safari scroll lock pattern with position:fixed
- [Aleksandr Hovhannisyan: How to Open and Close HTML Dialogs](https://www.aleksandrhovhannisyan.com/blog/how-to-open-and-close-html-dialogs/) - Backdrop click detection patterns
- [Jared Cunha: HTML Dialog Accessibility](https://jaredcunha.com/blog/html-dialog-getting-accessibility-and-ux-right) - Focus management, scrollbar-gutter approach
- [Web Platform Features: dialog closedby](https://web-platform-dx.github.io/web-features-explorer/features/dialog-closedby/) - closedby attribute browser support status

### Tertiary (LOW confidence)
- [Jay Freestone: Locking Body Scroll iOS](https://www.jayfreestone.com/writing/locking-body-scroll-ios/) - Additional iOS scroll lock patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - `<dialog>` is a web standard, baseline since 2022, verified via MDN
- Architecture: HIGH - Patterns verified across multiple authoritative sources; existing codebase template system understood from reading source
- Pitfalls: HIGH - iOS Safari scroll lock and backdrop click are well-documented cross-browser issues confirmed by multiple sources

**Research date:** 2026-02-06
**Valid until:** 2026-05-06 (stable web APIs, unlikely to change)
