"use client";

import { useState } from "react";
import { CornerDownLeft, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { FileIcon } from "@/utils/icons/file";
import { AiIcon } from "@/utils/icons/ai";

export type ComposeMode = "reply" | "reply-all" | "forward" | "compose";

interface MailEditContainerProps {
  mode: ComposeMode;
  initialTo?: string;
  initialCc?: string;
  initialBcc?: string;
  initialSubject?: string;
  initialBody?: string;
  onClose: () => void;
  onSend?: (data: {
    to: string;
    cc: string;
    bcc: string;
    subject: string;
    body: string;
  }) => void;
}

export function MailEditContainer({
  mode,
  initialTo = "",
  initialCc = "",
  initialBcc = "",
  initialSubject = "",
  initialBody = "",
  onClose,
  onSend,
}: MailEditContainerProps) {
  const [showCc, setShowCc] = useState(!!initialCc || mode === "reply-all");
  const [showBcc, setShowBcc] = useState(!!initialBcc);
  const [composeTo, setComposeTo] = useState(initialTo);
  const [composeCc, setComposeCc] = useState(initialCc);
  const [composeBcc, setComposeBcc] = useState(initialBcc);
  const [composeSubject, setComposeSubject] = useState(initialSubject);
  const [composeBody, setComposeBody] = useState(initialBody);

  const handleSend = () => {
    onSend?.({
      to: composeTo,
      cc: composeCc,
      bcc: composeBcc,
      subject: composeSubject,
      body: composeBody,
    });
  };

  const getModeTitle = () => {
    switch (mode) {
      case "reply":
        return "Reply";
      case "reply-all":
        return "Reply All";
      case "forward":
        return "Forward";
      case "compose":
        return "New Message";
      default:
        return "Compose";
    }
  };

  return (
    <div className="border-t border-border bg-zinc-900">
      <div className="p-4 space-y-3">
        {/* To Field */}
        <div className="flex items-center gap-2 border-b border-border/40 pb-2">
          <label className="text-sm text-muted-foreground min-w-12">To:</label>
          <Input
            value={composeTo}
            onChange={(e) => setComposeTo(e.target.value)}
            placeholder="Recipient email"
            className="border-0 bg-transparent focus-visible:ring-0 shadow-none px-0"
          />
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="sm"
              className="h-6 text-xs text-muted-foreground hover:text-foreground"
              onClick={() => setShowCc(!showCc)}
            >
              Cc
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 text-xs text-muted-foreground hover:text-foreground"
              onClick={() => setShowBcc(!showBcc)}
            >
              Bcc
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={onClose}
            >
              <X className="h-3 w-3" />
            </Button>
          </div>
        </div>

        {/* Cc Field */}
        {showCc && (
          <div className="flex items-center gap-2 border-b border-border/40 pb-2">
            <label className="text-sm text-muted-foreground min-w-12">
              Cc:
            </label>
            <Input
              value={composeCc}
              onChange={(e) => setComposeCc(e.target.value)}
              placeholder="Cc recipients"
              className="border-0 bg-transparent focus-visible:ring-0 shadow-none px-0"
            />
          </div>
        )}

        {/* Bcc Field */}
        {showBcc && (
          <div className="flex items-center gap-2 border-b border-border/40 pb-2">
            <label className="text-sm text-muted-foreground min-w-12">
              Bcc:
            </label>
            <Input
              value={composeBcc}
              onChange={(e) => setComposeBcc(e.target.value)}
              placeholder="Bcc recipients"
              className="border-0 bg-transparent focus-visible:ring-0 shadow-none px-0"
            />
          </div>
        )}

        {/* Message Body */}
        <div className="">
          <Textarea
            value={composeBody}
            onChange={(e) => setComposeBody(e.target.value)}
            placeholder="Write your message..."
            className="min-h-[130px] max-h-[400px] overflow-y-auto resize-none bg-transparent focus-visible:ring-0 shadow-none"
          />
        </div>

        {/* Actions */}
        <div className="flex items-center justify-between">
          <div>
            <Button variant="ghost" size="sm" className="text-muted-foreground">
              <AiIcon className=" h-4 w-4" /> Generate
            </Button>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" className="text-muted-foreground">
              <FileIcon className=" h-4 w-4 text-muted-foreground" /> Attachment
            </Button>
            <Button onClick={handleSend} size="sm">
              Send <CornerDownLeft />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
