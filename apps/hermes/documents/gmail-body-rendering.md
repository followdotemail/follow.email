## Gmail Body Parsing & Rendering Pipeline

This document captures, in exhaustive detail, how Gmail message bodies flow from Google’s API all the way to the Zero frontend, including every transformation layer that impacts their HTML structure, sanitization, and final rendering.

---

### 1. Gmail Thread Fetch (`GoogleMailManager.get`)

**Location**: `apps/server/src/lib/driver/google.ts`

1. **Thread retrieval**
   - Calls `gmail.users.threads.get` with `format: 'full'`.
   - Provides `quotaUser` (email + env) so Google rate limits per user/environment.

2. **Body extraction per message**
   - Prefers `message.payload.body.data`.
   - Otherwise calls recursive `findHtmlBody(parts)` to locate the first `text/html` MIME section.
   - Falls back to the first part’s body data if nothing else is available.

3. **Decoding & heuristics**
   - Base64-url data is decoded with `fromBinary` (internally replaces `-_/` variants, then `TextDecoder`).
   - The repo uses `he.decode` to convert HTML entities.
   - It heuristically detects plain-text bodies by stripping tags and comparing to raw text; plain text gets `<br>` inserted for new lines.
   - Result stored in `decodedBody`.

4. **Inline CID image inlining**
   - Scans `payload.parts` for inline attachments (`Content-Disposition: inline` + `Content-ID` header).
   - For each match:
     - Downloads the attachment bytes via `getAttachment`.
     - Cleans `content-id` (removes `< >`), escapes regex characters, replaces every `cid:...` reference inside `processedBody` with `data:<mime>;base64,<payload>`.
   - Updates `processedBody` (ultimately saved as `decodedBody`).

5. **Metadata assembly**
   - Delegates to `parse(message)` to pull headers (subject, sender, recipients, TLS, reply/meta IDs, labels).
   - Collates attachment metadata via `findAttachments` but intentionally leaves `body: ''` (attachments lazily fetched later).
   - Builds the final `ParsedMessage` subset:
     - `body`, `processedHtml`, `blobUrl` intentionally blank.
     - Inline result lives in `decodedBody`.
     - Attachments include filenames, MIME types, sizes, header echo, attachmentId handles.

6. **Thread response**
   - Aggregates unread flags, label union, latest non-draft message, reply counts, etc., and returns to TRPC.

---

### 2. HTML Sanitization (`processEmailContent` Mutation)

**Entry point**: `apps/server/src/trpc/routes/mail.ts`

1. **Request**
   - Frontend calls `trpc.mail.processEmailContent` with raw `decodedBody`, `shouldLoadImages` (based on trust & settings), and the active theme.

2. **Server handler**
   - Invokes `processEmailHtml` from `apps/server/src/lib/email-processor.ts`.

3. **`preprocessEmailHtml` steps**
   - Runs `sanitize-html` with expanded allowed tags (`img`, `style`, `details`, etc.) and attributes (dimensions, table layout).
   - Forces safe schemes (`http`, `https`, `mailto`, `tel`, `data`, `cid`) and `target="_blank" / rel="noopener noreferrer"` for links.
   - Parses the sanitized DOM via Cheerio:
     - Iterates `<style>` blocks, filtering CSS with `@barkleapp/css-sanitizer` (allows only core properties; strips `@import`, `url()`, `expression`, etc.).
     - Wraps `blockquote` and `.gmail_quote` elements in `<details class="quoted-toggle">` with a `<summary>` toggle to collapse quoted replies.
     - Removes `<title>` tags, 1×1 & 0×0 tracking pixels, and invisible preheader snippets (based on class names + inline styles).

4. **`applyEmailPreferences` steps**
   - Loads the sanitized HTML back into Cheerio.
   - When `shouldLoadImages` is `false`, replaces each `<img src="...">` (excluding `cid:` inline assets) with a hidden span placeholder and sets `hasBlockedImages = true`.
   - Prepends a `<style>` block tailored to the requested theme (`light` or `dark`), applying:
     - Shadow host defaults (`display:block`, background/text color).
     - Reset CSS (`box-sizing`, `body` margin/padding).
     - Link colors per theme.
     - `details.quoted-toggle` accent colors & behavior.
   - Emits `{ processedHtml, hasBlockedImages }`.

---

### 3. Frontend Rendering (`MailContent` + `MailDisplay`)

**Key files**:
- `apps/mail/components/mail/mail-display.tsx`
- `apps/mail/components/mail/mail-content.tsx`

1. **Component hierarchy**
   - `ThreadDisplay` maps each `ParsedMessage` to `MailDisplay`.
   - `MailDisplay` surfaces metadata (avatar, sender, recipients, timeline) and, if `decodedBody` exists, renders `<MailContent id html senderEmail />`.

2. **`MailContent` behavior**
   - Accesses user & tenant settings to decide whether the sender is trusted or temporary remote images are allowed.
   - Uses `useQuery` to call `trpc.mail.processEmailContent`, keyed by message ID, trust flag, and theme.
   - Receives sanitized HTML plus `hasBlockedImages` boolean; exposes the latter via UI state (CSP warning toggle).
   - Creates a Shadow DOM host (`div` with `ref`), calling `attachShadow({ mode: 'open' })` once.
   - On every processed-html update, writes directly to `shadowRoot.innerHTML`, ensuring email CSS cannot bleed into the app shell.
   - Registers a `capture` event listener for `<img>` errors to hide blocked images and flip the CSP flag if untrusted.

3. **Attachments & printing**
   - Below the `MailContent` block, `MailDisplay` lists per-message attachments; separate hooks retrieve file blobs on demand.
   - `ThreadDisplay` includes a print helper that reuses `cleanHtml(message.decodedBody)` to dump sanitized HTML into a print-only iframe, ensuring parity with the on-screen content.

---

### 4. Summary of Data Contracts

| Stage | Field | Notes |
| --- | --- | --- |
| Gmail API response | `payload.body.data` / `payload.parts` | Base64-url encoded raw MIME segments. |
| Driver output (`ParsedMessage`) | `decodedBody` | HTML (possibly text converted to HTML) after inline CID replacement. |
| TRPC `processEmailContent` | `processedHtml`, `hasBlockedImages` | Sanitized HTML + image-block flag based on user prefs. |
| Frontend state | Shadow DOM inner HTML | Exact markup inserted into the component’s shadow root for isolated rendering. |

This layered approach ensures:
1. **Safety** – All remote HTML is sanitized and optionally stripped of tracking pixels before reaching the browser.
2. **Fidelity** – Inline images, tables, and marketing layouts are preserved thanks to minimal-yet-strict CSS allowances.
3. **User control** – Image loading & theming respect per-sender trust decisions, enforced server-side and client-side.
4. **Isolation** – Shadow DOM encapsulates third-party styles, preventing conflicts with Zero’s design system.

For any modifications, touchpoints span Gmail driver parsing, email processor sanitation rules, TRPC mutation contracts, and the `MailContent` renderer. Always validate the end-to-end flow after changes to ensure message fidelity and security posture remain intact.

