"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

type EmailPreviewProps = {
  html: string;
  className?: string;
  initialWidth?: number;
};

export function EmailPreview({
  html,
  className,
  initialWidth = 700,
}: EmailPreviewProps) {
  const [width, setWidth] = React.useState<number>(initialWidth);
  const [height, setHeight] = React.useState<number>(900);
  const iframeRef = React.useRef<HTMLIFrameElement>(null);

  // Measure content height after load so the frame fits the email
  const updateHeight = React.useCallback(() => {
    const iframe = iframeRef.current;
    if (!iframe) return;
    try {
      const doc = iframe.contentDocument || iframe.contentWindow?.document;
      if (!doc) return;
      const nextHeight =
        doc.documentElement?.scrollHeight || doc.body?.scrollHeight || height;
      setHeight(Math.max(400, Math.min(nextHeight, 3000)));
    } catch {
      // If sandbox prevents access, fall back to a reasonable default
      setHeight(1000);
    }
  }, [height]);

  return (
    <div className={cn("w-full space-y-4 bg-black", className)}>
      <div className="flex w-full justify-center">
        <div
          className="overflow-hidden"
          style={{ width: `100%` }}
          aria-label="Email preview frame container"
        >
          <iframe
            ref={iframeRef}
            title="Email preview"
            sandbox="allow-same-origin"
            srcDoc={html}
            style={{ width: "100%", height }}
            className="block bg-black"
            onLoad={updateHeight}
          />
        </div>
      </div>
    </div>
  );
}

export default EmailPreview;
