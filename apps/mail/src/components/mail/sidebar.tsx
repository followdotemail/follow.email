"use client";

import * as React from "react";
import { Pencil } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "../ui/button";
import { AccountSwitcher } from "./account-switcher";
import { Nav } from "./nav";
import { useAuth } from "@clerk/nextjs";
import AppLogo from "../app-logo";
import { UserProfile } from "./user-profile";
import { ComposeDialog } from "./compose-dialog";

type IconComponent = React.ComponentType<React.SVGProps<SVGSVGElement>>;

interface SidebarProps {
  accounts: {
    label: string;
    email: string;
    icon: React.ReactNode;
  }[];
  isCollapsed: boolean;
  navItems: {
    main: {
      title: string;
      label?: string;
      icon: IconComponent;
      variant: "default" | "ghost";
    }[];
    secondary: {
      title: string;
      label?: string;
      icon: IconComponent;
      variant: "default" | "ghost";
    }[];
  };
}

export function Sidebar({ accounts, isCollapsed, navItems }: SidebarProps) {
  const { isLoaded, isSignedIn, getToken, sessionId } = useAuth();
  const [copied, setCopied] = React.useState(false);
  const [composeOpen, setComposeOpen] = React.useState(false);

  const copyTokenToClipboard = async () => {
    try {
      const token = await getToken();
      if (token) {
        await navigator.clipboard.writeText(token);
        setCopied(true);
        window.setTimeout(() => setCopied(false), 1500);
      }
    } catch (error) {
      console.error("Error copying token:", error);
    }
  };
  return (
    <section className="h-full flex flex-col justify-between">
      <div>
        <div
          className={cn(
            "flex h-[52px] items-center justify-start gap-1",
            isCollapsed ? "h-[52px] justify-center" : "px-2"
          )}
        >
          <AppLogo /> {isCollapsed ? "" : "Follow Email"}
        </div>
        <div
          className={cn(
            "flex h-[52px] items-center justify-center",
            isCollapsed ? "h-[52px]" : "px-2"
          )}
        >
          <Button
            variant="blue"
            className={cn("w-full", isCollapsed && "w-fit aspect-square")}
            size={isCollapsed ? "icon" : "default"}
            onClick={() => setComposeOpen(true)}
          >
            <Pencil className="h-3 w-3" /> {isCollapsed ? "" : "New Mail"}
          </Button>
        </div>
        <Nav isCollapsed={isCollapsed} links={navItems.main} />
      </div>
      <div>
        <Button
          variant={"ghost"}
          className="w-full"
          onClick={copyTokenToClipboard}
          disabled={!isSignedIn || !isLoaded || copied}
        >
          {copied ? "Copied" : "Token"}
        </Button>
        {/* <Nav isCollapsed={isCollapsed} links={navItems.secondary} /> */}
        <div className={cn("px-2", isCollapsed && "px-0 mb-4")}>
          <UserProfile isCollapsed={isCollapsed} />
        </div>
      </div>
      <ComposeDialog open={composeOpen} onOpenChange={setComposeOpen} />
    </section>
  );
}
