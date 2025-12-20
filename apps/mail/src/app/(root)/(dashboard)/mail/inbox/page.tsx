"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import { useRouter } from "next/navigation";
import useSWR from "swr";
import { accounts } from "@/constants/mail-data";
import { Mail } from "@/components/mail/mail";
import { mailListsFetcher, type MailListResponse } from "@/server/api/mail-lists";
import { EmailData } from "@/types/mail";

export default function MailPage() {
  const { getToken, isLoaded } = useAuth();
  const router = useRouter();
  const [token, setToken] = useState<string | null>(null);
  const [layout, setLayout] = useState<unknown[] | undefined>();
  const [collapsed, setCollapsed] = useState<unknown[] | undefined>();

  useEffect(() => {
    if (!isLoaded) return;

    const initAuth = async () => {
      const authToken = await getToken();
      if (!authToken) {
        router.push("/sign-in");
        return;
      }
      setToken(authToken);
    };

    initAuth();
  }, [isLoaded, getToken, router]);

  useEffect(() => {
    if (typeof window === "undefined") return;

    try {
      const layoutValue = localStorage.getItem(
        "react-resizable-panels:layout:mail",
      );
      const collapsedValue = localStorage.getItem(
        "react-resizable-panels:collapsed",
      );

      if (layoutValue) setLayout(JSON.parse(layoutValue));
      if (collapsedValue) setCollapsed(JSON.parse(collapsedValue));
    } catch (error) {
      console.error("Error parsing layout preferences:", error);
    }
  }, []);

  const { data, isLoading, error } = useSWR<MailListResponse>(
    token ? [token, 1, 20] : null,
    (key: unknown[]) => {
      const [authToken, page, limit] = key as [string, number, number];
      return mailListsFetcher(authToken, page, limit);
    },
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: true,
      dedupingInterval: 60000,
    },
  );

  if (!token) {
    return null;
  }

  // if (isLoading) {
  //   return <div className="flex items-center justify-center p-4">Loading...</div>;
  // }

  if (error) {
    return (
      <div className="flex items-center justify-center p-4 text-red-500">
        Error loading emails
      </div>
    );
  }

  const mails = (data?.emails || []) as EmailData[];

  return (
    <div className="flex-col md:flex">
      <Mail
        accounts={accounts}
        mails={mails}
        initialPagination={data?.pagination}
        defaultLayout={layout as number[] | undefined}
        defaultCollapsed={collapsed as boolean | undefined}
        navCollapsedSize={4}
      />
    </div>
  );
}
