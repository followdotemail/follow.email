import { ComponentProps } from "react";
import { formatDistanceToNow } from "date-fns/formatDistanceToNow";

import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Mail } from "@/constants/mail-data";
import { useMail } from "@/store/use-mail";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";
import { FileIcon } from "@/utils/icons/file";

interface MailListProps {
  items: Mail[];
}

export function MailList({ items }: MailListProps) {
  const [mail, setMail] = useMail();

  return (
    <ScrollArea className="h-[calc(100vh-60px)] overflow-y-auto">
      <div className="flex flex-col gap-2 p-2">
        {items.map((item) => (
          <button
            key={item.id}
            className={cn(
              "flex flex-col cursor-pointer items-start gap-2 rounded-lg p-2.5 sm:p-3 text-left text-sm transition-all hover:bg-accent active:bg-accent",
              mail.selected === item.id && "bg-muted"
            )}
            onClick={() =>
              setMail({
                ...mail,
                selected: item.id,
              })
            }
          >
            <div className="flex w-full items-center gap-3">
              <Avatar className="h-8 w-8 sm:h-10 sm:w-10">
                <AvatarImage alt={item.name} className="" />
                <AvatarFallback className="font-bold text-muted-foreground bg-zinc-700/60 text-base  ">
                  {item.name[0]}
                </AvatarFallback>
              </Avatar>
              <div className="flex w-full flex-col gap-1">
                <div className="flex items-center">
                  <div className="flex items-center gap-2 flex-1 min-w-0">
                    <div
                      className={cn(
                        "font-bold line-clamp-1 sm:text-sm",
                        !item.read ? "text-foreground" : "text-muted-foreground"
                      )}
                    >
                      {item.name}
                    </div>
                    {!item.read && (
                      <span className="flex h-2 w-2 rounded-full bg-yellow-500 flex-shrink-0" />
                    )}
                  </div>
                  <div
                    className={cn(
                      "ml-auto text-xs whitespace-nowrap flex-shrink-0 text-muted-foreground"
                    )}
                  >
                    {new Date(item.date).toISOString().split("T")[0]}
                  </div>
                </div>
                <div className="text-xs sm:text-sm text-muted-foreground line-clamp-1 flex items-center">
                  <FileIcon className="text-blue-400 size-4 mr-1" /> {item.subject}
                </div>
              </div>
            </div>

            {/* {item.labels.length ? (
              <div className="flex items-center gap-1.5 sm:gap-2 flex-wrap">
                {item.labels.map((label) => (
                  <Badge
                    key={label}
                    variant={getBadgeVariantFromLabel(label)}
                    className="px-1.5 sm:px-2 py-0.5 text-xs"
                  >
                    {label}
                  </Badge>
                ))}
              </div>
            ) : null} */}
          </button>
        ))}
      </div>
    </ScrollArea>
  );
}

function getBadgeVariantFromLabel(
  label: string
): ComponentProps<typeof Badge>["variant"] {
  if (["work"].includes(label.toLowerCase())) {
    return "default";
  }

  if (["personal"].includes(label.toLowerCase())) {
    return "outline";
  }

  return "secondary";
}
