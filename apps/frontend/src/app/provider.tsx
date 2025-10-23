"use client";
import { swrConfig } from "@/lib/swr-config";
import { SWRConfig } from "swr";

export function Provider({ children }: { children: React.ReactNode }) {
  return <SWRConfig value={swrConfig}>{children}</SWRConfig>;
}
