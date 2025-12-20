"use client";
import { swrConfig } from "@/lib/swr-config";
import { SWRConfig } from "swr";
import { NuqsAdapter } from "nuqs/adapters/next/app";

export function Provider({ children }: { children: React.ReactNode }) {
  return (
    <NuqsAdapter>
      <SWRConfig value={swrConfig}>{children}</SWRConfig>
    </NuqsAdapter>
  );
}
