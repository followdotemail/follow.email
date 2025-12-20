"use client";

import * as React from "react";
import { Menu, RefreshCw, Search } from "lucide-react";
import { useAuth } from "@clerk/nextjs";
import useSWR from "swr";

import { cn } from "@/lib/utils";
import { mailListsFetcher, type MailListResponse } from "@/server/api/mail-lists";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "../ui/resizable";
import { TooltipProvider } from "../ui/tooltip";
import { MailDisplay } from "@/components/mail/mail-display";
import { MailList } from "@/components/mail/mail-list";
import { Sidebar } from "@/components/mail/sidebar";
import { useMail } from "@/store/use-mail";

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
import { navItems } from "@/constants/sidebar";
import { EmailData, MailProps, Pagination } from "@/types/mail";



export function Mail({
  accounts,
  mails,
  initialPagination,
  defaultLayout = [20, 32, 48],
  defaultCollapsed = false,
  navCollapsedSize,
}: MailProps) {
  const [isCollapsed, setIsCollapsed] = React.useState(defaultCollapsed);
  const [mail] = useMail();
  const [isNavOpen, setIsNavOpen] = React.useState(false);
  const [emailCategory, setEmailCategory] = React.useState("primary");
  const [currentPage, setCurrentPage] = React.useState(1);
  const [allEmails, setAllEmails] = React.useState<EmailData[]>(mails);
  const [pagination, setPagination] = React.useState<Pagination | undefined>(
    initialPagination,
  );
  const { getToken } = useAuth();

  // Sync emails when mails prop changes (from parent revalidation)
  React.useEffect(() => {
    if (currentPage === 1) {
      setAllEmails(mails);
      setPagination(initialPagination);
    }
  }, [mails, initialPagination, currentPage]);

  // Use SWR for loading more emails
  const { isLoading: isLoadingMore } = useSWR<MailListResponse>(
    pagination?.has_next && currentPage > 1 && getToken
      ? [`mails-page-${currentPage}`, currentPage, pagination.limit, getToken]
      : null,
    async ([, page, limit, getTokenFn]: [string, number, number, () => Promise<string | null>]) => {
      const token = await getTokenFn();
      if (!token) {
        throw new Error("Not authenticated");
      }
      return mailListsFetcher(token, page, limit);
    },
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
      dedupingInterval: 60000,
      onSuccess: (data) => {
        if (data?.emails) {
          setAllEmails((prev) => [...prev, ...(data.emails as EmailData[])]);
          setPagination(data.pagination);
        }
      },
    },
  );

  const handleLoadMore = React.useCallback(() => {
    if (isLoadingMore) return;
    if (!pagination || !pagination.has_next) return;
    setCurrentPage((prev) => prev + 1);
  }, [isLoadingMore, pagination]);

  const handleRefresh = React.useCallback(async () => {
    if (!getToken) return;
    
    try {
      const token = await getToken();
      if (!token) return;

      const data = await mailListsFetcher(token, 1, pagination?.limit || 20);
      setAllEmails(data.emails as EmailData[]);
      setPagination(data.pagination);
      setCurrentPage(1);
    } catch (error) {
      console.error("Error refreshing emails:", error);
    }
  }, [getToken, pagination?.limit]);

  const selectedMail = React.useMemo<EmailData | null>(
    () => allEmails.find((item) => item.id === mail.selected) || null,
    [allEmails, mail.selected],
  );

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
                  <Button
                    variant={"ghost"}
                    size="icon"
                    onClick={handleRefresh}
                    disabled={!getToken}
                  >
                    <RefreshCw className="text-muted-foreground" />
                  </Button>
                </div>
              </div>

              <div>
                <MailList
                  items={allEmails}
                  hasMore={pagination?.has_next}
                  isLoadingMore={isLoadingMore}
                  onLoadMore={handleLoadMore}
                />
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

              <MailList
                items={allEmails}
                hasMore={pagination?.has_next}
                isLoadingMore={isLoadingMore}
                onLoadMore={handleLoadMore}
              />
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
