import AppLogo from "@/components/app-logo";
import Link from "next/link";
import React from "react";

const AuthLayout = ({ children }: { children: React.ReactNode }) => {
  return (
    <div className="bg-background flex min-h-svh flex-col items-center justify-center gap-6">
      <div className="relative flex w-full max-w-sm flex-col items-center gap-4">
        <div className="fixed inset-x-0 -top-52 m-auto aspect-video max-w-lg bg-gradient-to-tr from-yellow-400 to-violet-400 opacity-40 blur-3xl md:inset-x-16" />
        
        {children}
      </div>
    </div>
  );
};

export default AuthLayout;