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
import { Mail } from "@/constants/mail-data";
import { useMail } from "@/store/use-mail";
import EmailPreview from "./email-preview";
import { BellIcon } from "@/utils/icons/bell";
import {
  MailEditContainer,
  type ComposeMode as ComposeModeLiteral,
} from "./mail-edit-container";

interface MailDisplayProps {
  mail: Mail | null;
}

type ComposeMode = ComposeModeLiteral | null;

interface ComposeData {
  to: string;
  cc: string;
  bcc: string;
  subject: string;
  body: string;
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

  const handleBack = () => {
    setMail((prev) => ({ ...prev, selected: "" }));
  };

  const handleReply = () => {
    if (!mail) return;
    setComposeMode("reply");
    setComposeData({
      to: mail.email,
      cc: "",
      bcc: "",
      subject: `Re: ${mail.subject}`,
      body: "",
    });
  };

  const handleReplyAll = () => {
    if (!mail) return;
    setComposeMode("reply-all");
    setComposeData({
      to: mail.email,
      cc: "",
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
      body: `\n\n---------- Forwarded message ---------\nFrom: ${mail.name} <${
        mail.email
      }>\nDate: ${
        mail.date ? format(new Date(mail.date), "PPP p") : ""
      }\nSubject: ${mail.subject}\n\n${mail.text}`,
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
                            <AvatarImage alt={mail.name} />
                            <AvatarFallback className="font-bold  text-white bg-zinc-600 text-xs flex items-center justify-center opacity-80">
                              {mail.name[0]}
                            </AvatarFallback>
                          </Avatar>
                          <span className="font-medium text-sm pr-2 line-clamp-1">
                            {mail.name}
                          </span>
                        </button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="start" className="w-72">
                        <div className="flex items-center gap-3 p-3">
                          <Avatar className="h-10 w-10">
                            <AvatarImage alt={mail.name} />
                            <AvatarFallback className="font-bold text-white bg-purple-600 text-base">
                              {mail.name[0]}
                            </AvatarFallback>
                          </Avatar>
                          <div className="flex flex-col flex-1 min-w-0">
                            <div className="font-semibold text-base">
                              {mail.name}
                            </div>
                            <div className="text-sm text-muted-foreground truncate">
                              {mail.email}
                            </div>
                          </div>
                        </div>
                        <Separator />
                        <DropdownMenuItem
                          onClick={() => {
                            navigator.clipboard.writeText(mail.email);
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

                {mail.date && (
                  <div className="flex flex-col items-end gap-0.5 font-medium">
                    <div className="text-sm text-muted-foreground">
                      {format(new Date(mail.date), "MMM dd")}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {format(new Date(mail.date), "h:mm a")}
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
            <EmailPreview html={mail.text} />
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
