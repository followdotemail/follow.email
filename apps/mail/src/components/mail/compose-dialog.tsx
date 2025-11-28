"use client";

import * as React from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { MailEditContainer, type ComposeMode } from "./mail-edit-container";
import { useAuth } from "@clerk/nextjs";
import { BASE_URL } from "@/constants/base-url";

interface ComposeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ComposeDialog({ open, onOpenChange }: ComposeDialogProps) {
  const { getToken } = useAuth();
  const [sending, setSending] = React.useState(false);

  const handleSend = async (data: {
    to: string;
    cc: string;
    bcc: string;
    subject: string;
    body: string;
  }) => {
    setSending(true);
    try {
      const token = await getToken();
      if (!token) {
        throw new Error("Not authenticated");
      }

      // Parse email addresses (split by comma and trim)
      const toEmails = data.to
        .split(",")
        .map((email) => email.trim())
        .filter((email) => email.length > 0);
      const ccEmails = data.cc
        .split(",")
        .map((email) => email.trim())
        .filter((email) => email.length > 0);
      const bccEmails = data.bcc
        .split(",")
        .map((email) => email.trim())
        .filter((email) => email.length > 0);

      const response = await fetch(`${BASE_URL}/emails/send`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          to: toEmails,
          cc: ccEmails.length > 0 ? ccEmails : undefined,
          bcc: bccEmails.length > 0 ? bccEmails : undefined,
          subject: data.subject,
          body_text: data.body,
          body_html: data.body, // You might want to convert plain text to HTML
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to send email: ${response.status}`);
      }

      // Close dialog on success
      onOpenChange(false);
      
      // Optionally show success message or refresh email list
      console.log("Email sent successfully");
    } catch (error) {
      console.error("Error sending email:", error);
      alert(error instanceof Error ? error.message : "Failed to send email");
    } finally {
      setSending(false);
    }
  };

  const handleClose = () => {
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-hidden flex flex-col p-0">
        <DialogHeader className="px-6 pt-6 pb-2 border-b">
          <DialogTitle>New Message</DialogTitle>
        </DialogHeader>
        <div className="flex-1 overflow-y-auto">
          <MailEditContainer
            mode="compose"
            onClose={handleClose}
            onSend={handleSend}
            sending={sending}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}

