"use client";

import * as React from "react";
import { useUser, useClerk } from "@clerk/nextjs";
import { LogOut } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";
import { cn } from "@/lib/utils";

interface UserProfileProps {
  isCollapsed?: boolean;
}

export function UserProfile({ isCollapsed }: UserProfileProps) {
  const { user } = useUser();
  const { signOut } = useClerk();

  if (!user) return null;

  const userImageUrl = user.imageUrl;
  const userName = user.fullName || user.firstName || user.username || "User";
  const userEmail = user.primaryEmailAddress?.emailAddress || "";

  const initials = user.firstName?.[0] || user.username?.[0] || "U";

  const handleSignOut = () => {
    signOut();
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="mb-2 border" asChild>
        <button
          className={cn(
            "flex items-center cursor-pointer gap-2 rounded-md transition-colors",
            isCollapsed
              ? "justify-center p-2"
              : "w-full justify-start px-2 py-2"
          )}
        >
          <Avatar className="h-8 w-8">
            <AvatarImage src={userImageUrl} alt={userName} />
            <AvatarFallback>{initials}</AvatarFallback>
          </Avatar>
          {!isCollapsed && (
            <div className="flex flex-col items-start overflow-hidden">
              <span className="text-sm font-medium truncate max-w-[150px]">
                {userName}
              </span>
              <span className="text-xs text-muted-foreground truncate max-w-[150px]">
                {userEmail}
              </span>
            </div>
          )}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align={isCollapsed ? "start" : "end"}
        className="w-64"
      >
        <div className="flex items-center gap-3 p-3">
          <Avatar className="h-10 w-10">
            <AvatarImage src={userImageUrl} alt={userName} />
            <AvatarFallback>{initials}</AvatarFallback>
          </Avatar>
          <div className="flex flex-col overflow-hidden">
            <span className="text-sm font-semibold truncate">{userName}</span>
            <span className="text-xs text-muted-foreground truncate">
              {userEmail}
            </span>
          </div>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={handleSignOut}
          className="cursor-pointer dark:hover:bg-white/10 transition-all"
        >
          <LogOut className="mr-2 h-4 w-4" />
          <span>Log out</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

