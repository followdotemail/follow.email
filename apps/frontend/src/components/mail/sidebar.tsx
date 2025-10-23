"use client";

import * as React from "react";
import { Pencil } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "../ui/button";
import { AccountSwitcher } from "./account-switcher";
import { Nav } from "./nav";

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
  return (
    <>
      <div
        className={cn(
          "flex h-[52px] items-center justify-center",
          isCollapsed ? "h-[52px]" : "px-2"
        )}
      >
        <AccountSwitcher isCollapsed={isCollapsed} accounts={accounts} />
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
        >
          <Pencil className="h-3 w-3" /> {isCollapsed ? "" : "New Mail"}
        </Button>
      </div>
      <Nav isCollapsed={isCollapsed} links={navItems.main} />
      <Nav isCollapsed={isCollapsed} links={navItems.secondary} />
    </>
  );
}
