"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";
import React from "react";

type IconComponent = React.ComponentType<React.SVGProps<SVGSVGElement>>;

interface NavProps {
  isCollapsed: boolean;
  links: {
    title: string;
    label?: string;
    icon: IconComponent;
    variant: "default" | "ghost";
    href?: string;
  }[];
}

export function Nav({ links, isCollapsed }: NavProps) {
  const pathname = usePathname();
  
  const getHref = (link: typeof links[0]) => {
    if (link.href) return link.href;
    // Default route mapping based on title
    const routeMap: Record<string, string> = {
      Inbox: "/mail/inbox",
      Draft: "/mail/draft",
      Sent: "/mail/sent",
      Schedule: "/mail/schedule",
      Trash: "/mail/trash",
      Archive: "/mail/archive",
      Spam: "/mail/spam",
      Feedback: "/mail/feedback",
      Settings: "/mail/settings",
    };
    return routeMap[link.title] || "#";
  };

  const isActive = (link: typeof links[0]) => {
    const href = getHref(link);
    if (href === "#") return false;
    return pathname === href || pathname.startsWith(href + "/");
  };

  return (
    <div
      data-collapsed={isCollapsed}
      className="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2"
    >
      <nav className="grid gap-1 px-2 group-data-[collapsed=true]:justify-center group-data-[collapsed=true]:px-2">
        {links.map((link, index) => {
          const href = getHref(link);
          const active = isActive(link);
          
          return isCollapsed ? (
            <Tooltip key={index} delayDuration={0}>
              <TooltipTrigger asChild>
                <Link
                  href={href}
                  className={cn(
                    "flex items-center rounded-md px-2 py-2.5 text-sm font-normal transition-all",
                    active
                      ? "bg-white/10 dark:bg-white/10 backdrop-blur-md border border-white/10 shadow-lg"
                      : "hover:bg-muted hover:text-muted-foreground hover:dark:bg-muted hover:dark:text-muted-foreground active:bg-muted"
                  )}
                >
                  <link.icon className="h-4 w-4" />
                  <span className="sr-only">{link.title}</span>
                </Link>
              </TooltipTrigger>
              <TooltipContent side="right" className="flex items-center gap-4">
                {link.title}
                {link.label && (
                  <span className="ml-auto text-muted-foreground">
                    {link.label}
                  </span>
                )}
              </TooltipContent>
            </Tooltip>
          ) : (
            <Link
              key={index}
              href={href}
              className={cn(
                "flex items-center rounded-md px-3 py-2 text-sm font-normal transition-all touch-manipulation",
                active
                  ? "bg-white/10 dark:bg-white/10 backdrop-blur-md border border-white/10 shadow-lg"
                  : "hover:bg-muted hover:dark:bg-muted active:bg-muted"
              )}
            >
              <link.icon className="mr-3 h-4 w-4 shrink-0" />
              <span className="flex-1">{link.title}</span>

              {link.label && (
                <span className="ml-auto text-muted-foreground">
                  {link.label}
                </span>
              )}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
