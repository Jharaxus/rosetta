# Rosetta Design System — Direction D · Atelier

This skill provides the full UI/UX reference for the Rosetta app. Always consult it when building, modifying, or reviewing frontend components.

## Origin

Designed in Claude Design (claude.ai/design). The user chose **Direction D – Atelier** from two options. Key brief: language-learning app (FR → DE) for 25–30-year-old professionals (engineering/finance backgrounds). Tone: *studious but soft, private library at night — welcoming, not flashy, energetic enough to work*.

---

## Color Tokens

All tokens are in `frontend/src/styles/tokens.css` as CSS custom properties.

| Token | Value | Usage |
|-------|-------|-------|
| `--color-bg` | `#fbf6ec` | Page background — ivoire clair |
| `--color-paper` | `#ffffff` | Card surfaces, inputs |
| `--color-ink` | `#1a2238` | Primary text, primary button bg |
| `--color-ink2` | `#3a4258` | Secondary text, labels |
| `--color-dim` | `#7a7466` | Captions, muted text |
| `--color-dim2` | `#b0a896` | Placeholder, disabled states |
| `--color-gold` | `#b88a3a` | Warm gold — accent, active states, highlights |
| `--color-gold-soft` | `#e8d4a4` | Soft gold tint — icon backgrounds, hover |
| `--color-rule` | `rgba(26,34,56,0.08)` | Subtle dividers, card borders |
| `--color-rule-hi` | `rgba(26,34,56,0.18)` | Stronger borders, input borders |

### Palette Variants (switchable, not yet implemented in the app)

The design system supports three palette variants that only change `gold`, `goldSoft`, and `navy`:

- **Atelier** (default): gold `#b88a3a`, navy `#1a2238` — warm and restrained
- **Azur**: gold `#d49a3a`, navy `#1e3a78` — more energy, electric cobalt primary
- **Lagune**: gold `#c89a4a`, navy `#1a4a52` — teal accent, vintage atlas feel

---

## Typography

Fonts are loaded via Google Fonts in `frontend/index.html`.

| Role | Variable | Family | Notes |
|------|----------|--------|-------|
| Display | `--font-display` | `"Fraunces", "Source Serif 4", Georgia, serif` | Hero headings, card titles, italic accents on names |
| Body/UI | `--font-sans` | `"Manrope", "Inter Tight", system-ui, sans-serif` | All body text, buttons, nav, labels |
| Metadata | `--font-mono` | `"JetBrains Mono", monospace` | All-caps micro-labels (9–11px), letterSpacing 1.2–1.6 |

### Type Scale

| Use | Font | Size | Weight | Notes |
|-----|------|------|--------|-------|
| Hero greeting | display | 42–84px | 400 | letterSpacing -1 to -2.5; italic name in gold |
| Section heading | display | 24–30px | 400 | letterSpacing -0.4 |
| Card heading | display | 16–22px | 400 | |
| Section label | mono | 9–11px | 400–500 | ALL CAPS, letterSpacing 1.2–1.6, color: gold |
| Body copy | sans | 12–15px | 400 | lineHeight 1.5–1.6 |
| Button | sans | 13–15px | 500 | |
| Caption / meta | sans | 11–13px | 400 | color: dim |

---

## Spacing & Layout

| Token | Value | Variable |
|-------|-------|----------|
| Page h-padding | 28px (comfy) / 22px (compact) | — |
| Section gap | 26–36px | — |
| Card padding | 20–36px | — |
| Card radius | 20px | `--radius-card` |
| Card radius (sm) | 16px | `--radius-card-sm` |
| Button radius | 24px | `--radius-pill` |
| Input radius | 12px | `--radius-input` |
| Tag/chip radius | 8px | `--radius-tag` |

### Shadows

```css
--shadow-card:    0 1px 2px rgba(26,34,56,0.04), 0 12px 32px rgba(26,34,56,0.06);
--shadow-card-lg: 0 1px 3px rgba(26,34,56,0.04), 0 24px 48px rgba(26,34,56,0.06);
```

Very subtle. Two-layer (crisp near-shadow + soft far-shadow). Never use heavy drop shadows.

---

## Component Patterns

### Buttons

Three variants — all use pill shape (border-radius 24px), height 48–52px, Manrope 14–15px weight 500.

| Variant | Background | Text color | Border | Use case |
|---------|-----------|-----------|--------|----------|
| Primary | `--color-ink` (navy) | `--color-bg` (ivory) | none | Main CTA (sign in, next step) |
| Gold | `--color-gold` | `--color-ink` | none | Confirm/submit actions |
| Ghost | transparent | `--color-dim` | 1px `--color-rule-hi` | Back/skip/secondary |

Always include the italic serif arrow as a directional indicator for forward actions:
```tsx
<span style={{ fontFamily: 'var(--font-display)', fontStyle: 'italic' }}>→</span>
```

### Cards

- Background: `--color-paper` (white) on `--color-bg` (ivory) page
- Radius: `--radius-card` (20px) for main, `--radius-card-sm` (16px) for secondary
- Shadow: `--shadow-card`
- Padding: 24–28px mobile, 28–36px desktop
- Always have a `cardLabel` in mono gold above the content

### Inputs

- Height: 44px
- Border: `1px solid var(--color-rule-hi)` at rest, `var(--color-gold)` on focus
- Background: `--color-bg` at rest, `--color-paper` on focus
- Radius: `--radius-input` (12px)
- Error state: border `#c0442a`
- Error message: 11px sans, color `#c0442a`, below the input

### Section Labels (mono caps)

Used above cards and in top bars. Pattern:
```tsx
<div style={{
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  letterSpacing: 1.4,
  color: 'var(--color-gold)'
}}>
  LEÇON DU JOUR · 3 MIN
</div>
```

### Bottom / Top Navigation

- Active item: `--color-ink`, weight 600, gold 4×4px dot indicator
- Inactive: `--color-dim`, weight 400
- Font: sans, 11–13px

---

## Animations

Defined in `frontend/src/styles/tokens.css`.

### `fadeUp`
For page content entering the viewport. Apply with animation-delay staggering for depth:
```css
animation: fadeUp 0.7s ease-out both;
/* secondary elements */
animation: fadeUp 0.9s ease-out both;
animation-delay: 0.1s;
```

### `breathe`
For live/active indicators (e.g., "AI tutor is online" dot):
```css
animation: breathe 3.2s ease-in-out infinite;
```

---

## Page Layouts

### LandingPage (`/`)
- Full viewport centered column, max-width 400px
- Logo: gold circle dot (22px) + "Rosetta" in Fraunces italic 28px
- Tagline in Manrope dim, 15px
- Primary (navy) SSO button
- Ghost register link below

### RegisterPage (`/register`)
- White paper card (max-width 440px) centered on ivory page
- `stepLabel` in mono gold at top
- Fraunces 28px title
- 4 fields (name, email, password, confirm)
- Ghost "retour" + Gold "Créer mon compte →" buttons side by side
- Success screen: check mark in gold-soft circle, Fraunces italic greeting

### DashboardPage (`/dashboard`) — current scope
- Single-column, max-width 560px, left-aligned
- Top bar: logo mark + ghost logout button
- Hero greeting: mono label + large Fraunces title with italic gold name
- Assimil lesson card: progress bar + +/− stepper
- Profile link at bottom

---

## Tone & Copy (French)

- **Warm, conversational** — talks to the user like a knowledgeable friend, not an app
- **Names in italic Fraunces** — always italicize and gold-color user names: `<em>Élise</em>`
- **Directional arrow** — `→` in italic Fraunces (not emoji, not →  entity) for forward actions
- **Mono labels** in ALL CAPS with middle dots: `HISTOIRE DU JOUR · 3 MIN`
- **Numbers in italic display font** when prominent: `<em>47</em> jours de suite`
- **Breathing dot** for live AI tutor indicator (animated, gold)

### Sample copy register
| Screen | Key phrase |
|--------|-----------|
| Landing | "Apprenez l'allemand en dix minutes par jour." |
| Register title | "Créer un compte" |
| Register success | "Compte créé, *Élise*." |
| Dashboard greeting | "Bonjour, *Élise*." |
| Dashboard subtitle | "Continuez là où vous vous êtes arrêté." |

---

## File Locations

| File | Purpose |
|------|---------|
| `frontend/index.html` | Google Fonts links |
| `frontend/src/styles/tokens.css` | All CSS custom properties + keyframes |
| `frontend/src/styles/global.css` | Reset + base body styles |
| `frontend/src/pages/LandingPage.tsx` + `.module.css` | Sign-in page |
| `frontend/src/pages/RegisterPage.tsx` + `.module.css` | Sign-up form |
| `frontend/src/pages/DashboardPage.tsx` + `.module.css` | Logged-in home |

---

## Design Principles (Direction D summary)

1. **Generous whitespace** — fewer elements per screen, lots of breathing room
2. **Light ivory, not white** — the `#fbf6ec` background is warm, not clinical
3. **Gold is the accent, navy is the primary action color**
4. **Serif for emotional content, sans for UI chrome** — never swap them
5. **Soft shadows only** — the app should feel like paper, not glass
6. **Animations are subtle** — fadeUp 0.7s, no bounces, no spring physics
7. **Compact density available** — reduce padding/gaps by ~20% when on smaller screens
