"use client";

import * as React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import EmailPreview from "@/components/mail/email-preview";

type Props = {
  defaultHtml: string;
  subject?: string;
};

export default function EmailPreviewPlayground({
  defaultHtml,
  subject,
}: Props) {
  const [html, setHtml] = React.useState<string>(defaultHtml);
  const [url, setUrl] = React.useState<string>("");
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  async function handleFetch() {
    if (!url) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(url, { mode: "cors" });
      if (!res.ok) {
        throw new Error(`Request failed: ${res.status}`);
      }
      const text = await res.text();
      // We assume the response is HTML. If JSON with a "body" is returned, try to use it.
      try {
        const maybeJson = JSON.parse(text);
        if (maybeJson && typeof maybeJson.body === "string") {
          setHtml(maybeJson.body);
        } else {
          setHtml(text);
        }
      } catch {
        setHtml(text);
      }
    } catch (e: any) {
      // Note: Cross-origin requests may be blocked by CORS for some sites.
      setError(e?.message || "Failed to fetch the URL.");
    } finally {
      setLoading(false);
    }
  }

  function handleLoadSample() {
    setHtml(defaultHtml);
    setError(null);
  }

  function handleClear() {
    setHtml("");
    setError(null);
  }

  return (
    <div className="space-y-4">
      <div className="rounded-md border border-border bg-card p-4">
        <div className="grid gap-4 md:grid-cols-3">
          <div className="space-y-2 md:col-span-2">
            <Label htmlFor="email-html">Email HTML</Label>
            <Textarea
              id="email-html"
              value={html}
              onChange={(e) => setHtml(e.target.value)}
              rows={12}
              spellCheck={false}
              className="font-mono"
              placeholder="Paste your full email HTML here (including styles and tables)"
              aria-describedby="email-html-help"
            />
            <p id="email-html-help" className="text-xs text-muted-foreground">
              Paste raw HTML or fetch from a URL. Scripts won’t run; the preview
              is sandboxed.
            </p>
          </div>

          <div className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="email-url">Fetch from URL</Label>
              <Input
                id="email-url"
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://example.com/email.html"
              />
              <div className="flex gap-2">
                <Button onClick={handleFetch} disabled={!url || loading}>
                  {loading ? "Fetching..." : "Fetch URL"}
                </Button>
                <Button variant="outline" onClick={handleLoadSample}>
                  Load sample
                </Button>
                <Button variant="ghost" onClick={handleClear}>
                  Clear
                </Button>
              </div>
              {error ? (
                <p className="text-xs text-destructive" role="alert">
                  {error}
                </p>
              ) : null}
              {subject ? (
                <p className="mt-2 text-xs text-muted-foreground">
                  Subject: <span className="font-medium">{subject}</span>
                </p>
              ) : null}
            </div>
          </div>
        </div>
      </div>

      <EmailPreview html={html} initialWidth={600} />
    </div>
  );
}
