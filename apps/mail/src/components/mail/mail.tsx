"use client";

import * as React from "react";
import { Menu, RefreshCw, Search } from "lucide-react";

import { cn } from "@/lib/utils";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "../ui/resizable";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/tabs";
import { TooltipProvider } from "../ui/tooltip";
import { MailDisplay } from "@/components/mail/mail-display";
import { MailList } from "@/components/mail/mail-list";
import { Sidebar } from "@/components/mail/sidebar";
import { useMail } from "@/store/use-mail";
import { type Mail } from "@/constants/mail-data";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { InboxIcon } from "@/utils/icons/inbox";
import { DraftIcon } from "@/utils/icons/draft";
import { SentIcon } from "@/utils/icons/sent";
import { BinIcon } from "@/utils/icons/bin";
import { ArchiveIcon } from "@/utils/icons/archive";
import { SpamIcon } from "@/utils/icons/spam";
import { FeedbackIcon } from "@/utils/icons/feedback";
import { SettingIcon } from "@/utils/icons/setting";
import { ClockIcon } from "@/utils/icons/clock";

interface MailProps {
  accounts: {
    label: string;
    email: string;
    icon: React.ReactNode;
  }[];
  mails: Mail[];
  defaultLayout: number[] | undefined;
  defaultCollapsed?: boolean;
  navCollapsedSize: number;
}

const navItems = {
  main: [
    {
      title: "Inbox",
      label: "100",
      icon: InboxIcon,
      variant: "default" as const,
    },
    {
      title: "Draft",
      label: "2",
      icon: DraftIcon,
      variant: "ghost" as const,
    },
    {
      title: "Sent",
      label: "",
      icon: SentIcon,
      variant: "ghost" as const,
    },
    {
      title: "Schedule",
      label: "7",
      icon: ClockIcon,
      variant: "ghost" as const,
    },
    {
      title: "Trash",
      label: "5",
      icon: BinIcon,
      variant: "ghost" as const,
    },
    {
      title: "Archive",
      label: "",
      icon: ArchiveIcon,
      variant: "ghost" as const,
    },
    {
      title: "Spam",
      label: "24",
      icon: SpamIcon,
      variant: "ghost" as const,
    },
  ],
  secondary: [
    {
      title: "Feedback",
      label: "",
      icon: FeedbackIcon,
      variant: "ghost" as const,
    },
    {
      title: "Settings",
      label: "",
      icon: SettingIcon,
      variant: "ghost" as const,
    },
  ],
};

export function Mail({
  accounts,
  mails,
  defaultLayout = [20, 32, 48],
  defaultCollapsed = false,
  navCollapsedSize,
}: MailProps) {
  const [isCollapsed, setIsCollapsed] = React.useState(defaultCollapsed);
  const [mail] = useMail();
  const [isNavOpen, setIsNavOpen] = React.useState(false);
  const [emailCategory, setEmailCategory] = React.useState("primary");

  const selectedMail = mails.find((item) => item.id === mail.selected) || null;

  return (
    <TooltipProvider delayDuration={0}>
      {/* Tablet & Desktop Layout */}
      <div className="hidden lg:block">
        <ResizablePanelGroup
          direction="horizontal"
          onLayout={(sizes: number[]) => {
            document.cookie = `react-resizable-panels:layout:mail=${JSON.stringify(
              sizes
            )}`;
          }}
          className="h-full max-h-svh items-stretch"
        >
          <ResizablePanel
            defaultSize={defaultLayout[0]}
            collapsedSize={navCollapsedSize}
            collapsible={true}
            minSize={16}
            maxSize={16}
            onCollapse={() => {
              setIsCollapsed(true);
              document.cookie = `react-resizable-panels:collapsed=${JSON.stringify(
                true
              )}`;
            }}
            onResize={() => {
              setIsCollapsed(false);
              document.cookie = `react-resizable-panels:collapsed=${JSON.stringify(
                false
              )}`;
            }}
            className={cn(
              "p-1 pr-0",
              isCollapsed &&
                "min-w-[50px] transition-all duration-300 ease-in-out border-none"
            )}
          >
            <Sidebar
              accounts={accounts}
              isCollapsed={isCollapsed}
              navItems={navItems}
            />
          </ResizablePanel>
          <ResizableHandle withHandle className="boreder-none bg-transparent" />
          <ResizablePanel
            defaultSize={defaultLayout[1]}
            minSize={30}
            maxSize={30}
            className="p-1"
          >
            <div
              defaultValue="all"
              className="gap-0 bg-[#1A1A1A] rounded-xl border-none"
            >
              <div className="flex items-center px-3 py-2 space-x-3">
                <form className="w-full">
                  <div className="relative w-full bg-transparent">
                    <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                      placeholder="Search"
                      className="pl-8 w-full focus:outline-none border-none"
                    />
                  </div>
                </form>
                <Select value={emailCategory} onValueChange={setEmailCategory}>
                  <SelectTrigger className="w-[140px] h-8 text-sm border-none shadow-none focus:ring-0 text-muted-foreground">
                    <SelectValue
                      className="text-muted-foreground"
                      placeholder="Primary"
                    />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      className="text-muted-foreground"
                      value="primary"
                    >
                      Primary
                    </SelectItem>
                    <SelectItem
                      className="text-muted-foreground"
                      value="starred"
                    >
                      Starred
                    </SelectItem>
                    <SelectItem
                      className="text-muted-foreground"
                      value="promotions"
                    >
                      Promotions
                    </SelectItem>
                    <SelectItem
                      className="text-muted-foreground"
                      value="update"
                    >
                      Update
                    </SelectItem>
                  </SelectContent>
                </Select>
                <div>
                  <Button variant={"ghost"} size="icon">
                    <RefreshCw className="text-muted-foreground" />
                  </Button>
                </div>
              </div>

              <div>
                <MailList items={mails} />
              </div>
            </div>
          </ResizablePanel>
          <ResizablePanel
            defaultSize={defaultLayout[2]}
            minSize={30}
            className="p-1 pl-0"
          >
            <MailDisplay mail={selectedMail} />
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>

      {/* Mobile Layout */}
      <div className="lg:hidden">
        <div className="flex h-full flex-col">
          <div className="flex items-center gap-2  p-2">
            <Sheet open={isNavOpen} onOpenChange={setIsNavOpen}>
              <SheetTrigger asChild className="">
                <Button variant="ghost" size="icon">
                  <Menu className="h-5 w-5" />
                  <span className="sr-only">Toggle navigation</span>
                </Button>
              </SheetTrigger>
              <SheetContent side="left" className="w-[280px] p-0">
                <SheetHeader className="sr-only">
                  <SheetTitle>Navigation</SheetTitle>
                  <SheetDescription>Email navigation menu</SheetDescription>
                </SheetHeader>
                <Sidebar
                  accounts={accounts}
                  isCollapsed={false}
                  navItems={navItems}
                />
              </SheetContent>
            </Sheet>
            <form className="flex-1">
              <div className="relative">
                <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input placeholder="Search" className="pl-8" />
              </div>
            </form>
          </div>

          <div
            className={cn(
              "flex-1 overflow-hidden",
              selectedMail ? "hidden" : "block md:block"
            )}
          >
            <div defaultValue="all" className="h-full">
              <div className="flex items-center px-4 py-2 border-b">
                <Select value={emailCategory} onValueChange={setEmailCategory}>
                  <SelectTrigger className="w-[140px] h-8 text-sm border-none shadow-none">
                    <SelectValue placeholder="Primary" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="primary">Primary</SelectItem>
                    <SelectItem value="starred">Starred</SelectItem>
                    <SelectItem value="promotions">Promotions</SelectItem>
                    <SelectItem value="update">Update</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <MailList items={mails} />
            </div>
          </div>

          {/* Mobile Mail Display */}
          <div
            className={cn(
              "flex-1 overflow-hidden",
              selectedMail ? "block" : "hidden"
            )}
          >
            <MailDisplay mail={selectedMail} />
          </div>
        </div>
      </div>
    </TooltipProvider>
  );
}
