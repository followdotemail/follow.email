"use client";
import { ComponentProps } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useMail } from "@/store/use-mail";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";
import { FileIcon } from "@/utils/icons/file";

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

interface MailListProps {
  items: EmailData[];
}

export function MailList({ items }: MailListProps) {
  const [mail, setMail] = useMail();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  // Helper function to extract name from email
  const getNameFromEmail = (email: string) => {
    const match = email.match(/^(.+?)\s*<(.+)>$/);
    if (match) {
      return match[1].trim() || match[2].split("@")[0];
    }
    return email.split("@")[0];
  };

  // Helper function to parse labels
  const parseLabels = (labelsString: string) => {
    try {
      return JSON.parse(labelsString);
    } catch {
      return [];
    }
  };

  return (
    <ScrollArea className="h-[calc(100vh-60px)] overflow-y-auto">
      <div className="flex flex-col gap-2 p-2">
        {items.map((item) => {
          const senderName = getNameFromEmail(item.from_email);
          const labels = parseLabels(item.labels);

          return (
            <button
              key={item.id}
              className={cn(
                "flex flex-col cursor-pointer items-start gap-2 rounded-lg p-2.5 sm:p-3 text-left text-sm transition-all hover:bg-accent active:bg-accent",
                mail.selected === item.id && "bg-muted"
              )}
              onClick={() => {
                setMail({
                  ...mail,
                  selected: item.id,
                });
                const params = new URLSearchParams(searchParams?.toString());
                params.set("threadId", item.id);
                router.push(`${pathname}?${params.toString()}`, { scroll: false });
              }}
            >
              <div className="flex w-full items-center gap-3">
                <Avatar className="h-8 w-8 sm:h-10 sm:w-10">
                  <AvatarImage alt={senderName} className="" />
                  <AvatarFallback className="font-bold text-muted-foreground bg-zinc-700/60 text-base">
                    {senderName[0].toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="flex w-full flex-col gap-1">
                  <div className="flex items-center">
                    <div className="flex items-center gap-2 flex-1 min-w-0">
                      <div
                        className={cn(
                          "font-bold line-clamp-1 sm:text-sm",
                          !item.is_read
                            ? "text-foreground"
                            : "text-muted-foreground"
                        )}
                      >
                        {senderName}
                      </div>
                      {!item.is_read && (
                        <span className="flex h-2 w-2 rounded-full bg-yellow-500 flex-shrink-0" />
                      )}
                    </div>
                    <div className="ml-auto text-xs whitespace-nowrap flex-shrink-0 text-muted-foreground">
                      {new Date(item.updated_at).toISOString().split("T")[0]}
                    </div>
                  </div>
                  <div className="text-xs sm:text-sm text-muted-foreground line-clamp-1 flex items-center">
                    {item.has_attachments && (
                      <FileIcon className="text-blue-400 size-4 min-w-4 mr-1" />
                    )}
                    <span className="line-clamp-1">{item.subject}</span>
                  </div>
                </div>
              </div>
            </button>
          );
        })}
      </div>
    </ScrollArea>
  );
}