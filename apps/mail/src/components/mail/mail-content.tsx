"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { BASE_URL } from "@/constants/base-url";
import { Button } from "@/components/ui/button";
import type { Mail } from "@/constants/mail-data";

type MailContentProps = {
  mail: Mail | null;
};

type ProcessContentResponse = {
  processed_html: string;
  has_blocked_images: boolean;
};

export function MailContent({ mail }: MailContentProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const shadowRef = useRef<ShadowRoot | null>(null);
  const [processedHtml, setProcessedHtml] = useState("");
  const [hasBlockedImages, setHasBlockedImages] = useState(false);
  const [loadImages, setLoadImages] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchProcessedHtml = useCallback(
    (signal: AbortSignal) => {
      if (!mail?.text) {
        setProcessedHtml("");
        setHasBlockedImages(false);
        setError(null);
        return;
      }

      setLoading(true);
      setError(null);

      fetch(`${BASE_URL}/emails/process-content`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          html: mail.text,
          should_load_images: loadImages,
          theme: "dark",
        }),
        signal,
      })
        .then(async (res) => {
          if (!res.ok) {
            const message = await res.text();
            throw new Error(message || "Failed to process email content");
          }
          return res.json() as Promise<ProcessContentResponse>;
        })
        .then((data) => {
          setProcessedHtml(data.processed_html ?? "");
          setHasBlockedImages(Boolean(data.has_blocked_images));
        })
        .catch((err) => {
          if (err.name === "AbortError") return;
          setError(err.message);
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [mail?.text, loadImages, mail?.id],
  );

  useEffect(() => {
    if (hostRef.current && !shadowRef.current) {
      shadowRef.current = hostRef.current.attachShadow({ mode: "open" });
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    fetchProcessedHtml(controller.signal);
    return () => controller.abort();
  }, [fetchProcessedHtml]);

  useEffect(() => {
    if (!shadowRef.current) return;
    shadowRef.current.innerHTML = processedHtml || "";
  }, [processedHtml]);

  const toggleImages = () => setLoadImages((prev) => !prev);

  return (
    <div className="flex flex-col gap-3 p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="text-sm text-muted-foreground">
          {loading
            ? "Processing email content…"
            : hasBlockedImages && !loadImages
              ? "Remote images blocked"
              : "Content ready"}
        </div>
        <div className="flex gap-2">
          <Button
            size="sm"
            variant="secondary"
            className="bg-zinc-800"
            disabled={loading || !mail?.text}
            onClick={toggleImages}
          >
            {loadImages ? "Block Images" : "Load Images"}
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/60 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      <div className="min-h-[320px] rounded-xl border border-border/40 bg-background shadow-inner">
        <div ref={hostRef} className="block h-full w-full" />
      </div>
    </div>
  );
}

export default MailContent;

