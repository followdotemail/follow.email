import { format } from "date-fns/format";
import {
  Archive,
  ArchiveX,
  Copy,
  Forward,
  MailOpen,
  MoreVertical,
  Reply,
  ReplyAll,
  Trash2,
  X,
} from "lucide-react";
import { useState } from "react";
import { useQueryState } from "nuqs";
import { BASE_URL } from "@/constants/base-url";
import { useAuth } from "@clerk/nextjs";
import useSWR from "swr";
import { RiLoader3Fill, RiLoaderFill } from "react-icons/ri";


import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useMail } from "@/store/use-mail";

// Define the real email data structure
interface EmailData {
  id: string;
  clerk_id: string;
  message_id: string;
  thread_id: string;
  subject: string;
  from_email: string;
  from_name: string;
  to_emails: string;
  cc_emails: string;
  bcc_emails: string;
  updated_at: string;
  is_read: boolean;
  is_important: boolean;
  has_attachments: boolean;
  labels: string;
  last_sync_at: string;
}
import EmailPreview from "./email-preview";
import { BellIcon } from "@/utils/icons/bell";
import {
  MailEditContainer,
  type ComposeMode as ComposeModeLiteral,
} from "./mail-edit-container";

interface MailDisplayProps {
  mail: EmailData | null;
}

type ComposeMode = ComposeModeLiteral | null;

interface ComposeData {
  to: string;
  cc: string;
  bcc: string;
  subject: string;
  body: string;
}

interface EmailContent {
  attachments: any[];
  body: string;
  email_id: string;
}

export function MailDisplay({ mail }: MailDisplayProps) {
  const [, setMail] = useMail();
  const [composeMode, setComposeMode] = useState<ComposeMode>(null);
  const [composeData, setComposeData] = useState<ComposeData>({
    to: "",
    cc: "",
    bcc: "",
    subject: "",
    body: "",
  });
  
  const [threadId, setThreadId] = useQueryState("threadId");
  const { getToken } = useAuth();

  // Use SWR to fetch email content
  const { data: emailContent, error, isLoading: loading } = useSWR<EmailContent>(
    threadId && getToken
      ? [`email-content-${threadId}`, threadId, getToken]
      : null,
    async ([, threadId, getTokenFn]: [string, string, () => Promise<string | null>]) => {
      const token = await getTokenFn();
      if (!token) {
        throw new Error("Not authenticated");
      }

      const response = await fetch(`${BASE_URL}/emails/${threadId}`, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Server error: ${response.status}`);
      }

      return response.json();
    },
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: true,
      dedupingInterval: 60000,
      errorRetryCount: 2,
    },
  );

  // Helper function to extract name from email
  const getNameFromEmail = (email: string) => {
    const match = email.match(/^(.+?)\s*<(.+)>$/);
    if (match) {
      return match[1].trim() || match[2].split('@')[0];
    }
    return email.split('@')[0];
  };

  const handleBack = () => {
    setMail((prev) => ({ ...prev, selected: "" }));
    setThreadId(null);
  };

  const handleReply = () => {
    if (!mail) return;
    const senderEmail = mail.from_email.includes('<') 
      ? mail.from_email.match(/<(.+)>/)?.[1] || mail.from_email
      : mail.from_email;
    
    setComposeMode("reply");
    setComposeData({
      to: senderEmail,
      cc: "",
      bcc: "",
      subject: `Re: ${mail.subject}`,
      body: "",
    });
  };

  const handleReplyAll = () => {
    if (!mail) return;
    const senderEmail = mail.from_email.includes('<') 
      ? mail.from_email.match(/<(.+)>/)?.[1] || mail.from_email
      : mail.from_email;
    
    setComposeMode("reply-all");
    setComposeData({
      to: senderEmail,
      cc: mail.cc_emails || "",
      bcc: "",
      subject: `Re: ${mail.subject}`,
      body: "",
    });
  };

  const handleForward = () => {
    if (!mail) return;
    setComposeMode("forward");
    setComposeData({
      to: "",
      cc: "",
      bcc: "",
      subject: `Fwd: ${mail.subject}`,
      body: `\n\n---------- Forwarded message ---------\nFrom: ${getNameFromEmail(mail.from_email)} <${
        mail.from_email.includes('<') 
          ? mail.from_email.match(/<(.+)>/)?.[1] || mail.from_email
          : mail.from_email
      }>\nDate: ${
        mail.updated_at ? format(new Date(mail.updated_at), "PPP p") : ""
      }\nSubject: ${mail.subject}\n\n[Email content would be fetched from API]`,
    });
  };

  const handleCloseCompose = () => {
    setComposeMode(null);
    setComposeData({
      to: "",
      cc: "",
      bcc: "",
      subject: "",
      body: "",
    });
  };

  const handleSend = (data: ComposeData) => {
    // Here you would implement the actual send logic
    console.log("Sending email:", data);
    handleCloseCompose();
  };

  return (
    <div className="flex h-full flex-col bg-[#1A1A1A] rounded-xl overflow-hidden border-0">
      {mail && (
        <div className="flex items-center justify-between p-2">
          {/* Close button */}
          <Button
            variant="ghost"
            size="icon"
            className="mr-5"
            onClick={handleBack}
          >
            <X className="h-4 w-4" />
            <span className="sr-only">Close email</span>
          </Button>
          <div className="flex items-center gap-1 md:gap-2">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="hidden sm:flex">
                  <Archive className="h-4 w-4" />
                  <span className="sr-only">Archive</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Archive</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="hidden md:flex">
                  <ArchiveX className="h-4 w-4" />
                  <span className="sr-only">Move to spam</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Move to junk</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon">
                  <Trash2 className="h-4 w-4" />
                  <span className="sr-only">Move to trash</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Move to trash</TooltipContent>
            </Tooltip>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon">
                  <MoreVertical className="h-4 w-4" />
                  <span className="sr-only">More</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem>Mark as unread</DropdownMenuItem>
                <DropdownMenuItem>Star thread</DropdownMenuItem>
                <DropdownMenuItem>Add label</DropdownMenuItem>
                <DropdownMenuItem>Mute thread</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      )}

      {mail ? (
        <div className="flex flex-1 flex-col overflow-hidden">
          <div className="overflow-y-auto">
            {/* mail header */}
            <div className="flex flex-col gap-3 p-4 border-b border-border/40">
              {/* Subject Line */}
              <div className="sm:text-lg font-medium">{mail.subject}</div>

              {/* Sender Info Row */}
              <div className="flex items-start gap-3">
                <div className="flex flex-col flex-1 min-w-0">
                  {/* Category */}

                  <div className="flex items-center gap-2">
                    <div>
                      <Button
                        variant={"secondary"}
                        size={"icon"}
                        className="size-6 bg-violet-700 hover:bg-violet-600 rounded-sm"
                      >
                        <BellIcon className="mr-2 h-4 w-4" />
                      </Button>
                    </div>
                    <div className="bg-zinc-700 w-0.5 h-3.5" />
                    {/* chip */}
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <button className="flex items-center justify-start gap-2 border w-fit p-1 rounded-full hover:bg-accent transition-colors cursor-pointer">
                          <Avatar className="h-5 w-5">
                            <AvatarImage alt={getNameFromEmail(mail.from_email)} />
                            <AvatarFallback className="font-bold  text-white bg-zinc-600 text-xs flex items-center justify-center opacity-80">
                              {getNameFromEmail(mail.from_email)[0].toUpperCase()}
                            </AvatarFallback>
                          </Avatar>
                          <span className="font-medium text-sm pr-2 line-clamp-1">
                            {getNameFromEmail(mail.from_email)}
                          </span>
                        </button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="start" className="w-72">
                        <div className="flex items-center gap-3 p-3">
                          <Avatar className="h-10 w-10">
                            <AvatarImage alt={getNameFromEmail(mail.from_email)} />
                            <AvatarFallback className="font-bold text-white bg-purple-600 text-base">
                              {getNameFromEmail(mail.from_email)[0].toUpperCase()}
                            </AvatarFallback>
                          </Avatar>
                          <div className="flex flex-col flex-1 min-w-0">
                            <div className="font-semibold text-base">
                              {getNameFromEmail(mail.from_email)}
                            </div>
                            <div className="text-sm text-muted-foreground truncate">
                              {mail.from_email.includes('<') 
                                ? mail.from_email.match(/<(.+)>/)?.[1] || mail.from_email
                                : mail.from_email}
                            </div>
                          </div>
                        </div>
                        <Separator />
                        <DropdownMenuItem
                          onClick={() => {
                            const emailToCopy = mail.from_email.includes('<') 
                              ? mail.from_email.match(/<(.+)>/)?.[1] || mail.from_email
                              : mail.from_email;
                            navigator.clipboard.writeText(emailToCopy);
                          }}
                          className="cursor-pointer"
                        >
                          <Copy className="mr-2 h-4 w-4" />
                          Copy email address
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                    <button className="text-xs text-muted-foreground hover:underline cursor-pointer">
                      Details
                    </button>
                  </div>
                  <div className="text-sm text-muted-foreground p-1 font-medium">
                    To: You
                  </div>
                </div>

                {mail.updated_at && (
                  <div className="flex flex-col items-end gap-0.5 font-medium">
                    <div className="text-sm text-muted-foreground">
                      {format(new Date(mail.updated_at), "MMM dd")}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {format(new Date(mail.updated_at), "h:mm a")}
                    </div>
                  </div>
                )}

                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8">
                      <MoreVertical className="h-4 w-4" />
                      <span className="sr-only">More options</span>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem>Mark as unread</DropdownMenuItem>
                    <DropdownMenuItem>Star thread</DropdownMenuItem>
                    <DropdownMenuItem>Add label</DropdownMenuItem>
                    <DropdownMenuItem>Mark as spam</DropdownMenuItem>
                    <DropdownMenuItem>Print</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
            <Separator />
            {/* Email Preview */}
            <div className="p-4">
              {loading && (
                <div className="flex items-center justify-center py-8">
                  <div className="text-muted-foreground"><RiLoader3Fill size={20} className="animate-spin" />
                  </div>
                </div>
              )}
              
              {error && (
                <div className="text-red-500 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900 rounded-lg p-3 mb-4">
                  Error: {error instanceof Error ? error.message : "Failed to fetch email content"}
                </div>
              )}
              
              {emailContent && !loading && (
                <div className="prose prose-sm max-w-none dark:prose-invert">
                  <EmailPreview html={emailContent.body} />
                  {emailContent.attachments && emailContent.attachments.length > 0 && (
                    <div className="mt-4 pt-4 border-t border-border/40">
                      <h4 className="text-sm font-medium mb-2">Attachments ({emailContent.attachments.length})</h4>
                      <div className="space-y-2">
                        {emailContent.attachments.map((attachment, index) => (
                          <div key={index} className="flex items-center gap-2 text-sm text-muted-foreground">
                            <span>📎</span>
                            <span>{attachment.name || `Attachment ${index + 1}`}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
              
              {!emailContent && !loading && !error && (
                <div className="text-muted-foreground text-center py-8">
                  <p>No email content available</p>
                  <p className="text-xs mt-2">Thread ID: {mail.thread_id}</p>
                </div>
              )}
            </div>
            
            <div className="flex gap-2 p-4 border-t border-border/40">
              <Button
                variant={"secondary"}
                className="bg-zinc-800 px-2 py-1"
                size={"sm"}
                onClick={handleReply}
              >
                <Reply />
                Reply
              </Button>
              <Button
                variant={"secondary"}
                className="bg-zinc-800 px-2 py-1"
                size={"sm"}
                onClick={handleReplyAll}
              >
                <ReplyAll /> Reply All
              </Button>
              <Button
                variant={"secondary"}
                className="bg-zinc-800 px-2 py-1"
                size={"sm"}
                onClick={handleForward}
              >
                <Forward />
                Forward
              </Button>
            </div>
          </div>
          {/* Email Compose Section */}
          {composeMode && (
            <MailEditContainer
              mode={composeMode}
              initialTo={composeData.to}
              initialCc={composeData.cc}
              initialBcc={composeData.bcc}
              initialSubject={composeData.subject}
              initialBody={composeData.body}
              onClose={handleCloseCompose}
              onSend={handleSend}
            />
          )}
        </div>
      ) : (
        <div className="p-8 text-center text-muted-foreground min-h-svh flex flex-col items-center justify-center">
          <div>
            <MailOpen className="h-12 w-12" strokeWidth={1} />
          </div>
          <p className="mt-3">Choose an email to view details</p>
        </div>
      )}
    </div>
  );
}
