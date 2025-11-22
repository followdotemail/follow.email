"use client";

import AppLogo from "@/components/app-logo";
import { Button } from "@/components/ui/button";
import { BASE_URL } from "@/constants/base-url";
import { useAuth } from "@clerk/nextjs";

import { LoaderCircle, Mail } from "lucide-react";
import { useState } from "react";

export default function OnboardingClient() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { getToken } = useAuth();

  async function handleGmailConnect() {
    setLoading(true);
    setError(null);

    try {
      // Get the authentication token
      const token = await getToken();

      if (!token) {
        throw new Error("Not authenticated");
      }

      console.log("Initiating Gmail consent with token:", token);

      const response = await fetch(`${BASE_URL}/gmail/consent/initiate`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          return_url: `${window.location.origin}/mail`,
        }),
      });

      console.log("Gmail consent response:", response);
      console.log("Response status:", response.status);

      if (!response.ok) {
        throw new Error(`Server error: ${response.status}`);
      }

      const data = await response.json();
      console.log("Gmail consent data:", data);

      // Redirect to Google OAuth consent page
      if (data.auth_url) {
        console.log("Redirecting to auth URL:", data.auth_url);
        window.location.href = data.auth_url;
      } else {
        throw new Error(data.message || "No auth URL received from server");
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to connect to Gmail"
      );
      console.error(err);
    } finally {
      setLoading(false);
    }
  }
  return (
    <div className="min-h-svh flex flex-col items-center justify-center w-full px-4">
      <div className="max-w-md w-full text-center space-y-8">
        {/* Header Section */}
        <div className="space-y-3 mx-auto flex flex-col items-center justify-center">
          <AppLogo />
          <h1 className="text-2xl font-medium tracking-tight">
            Connect Your Mail
          </h1>
          <p className="text-muted-foreground text-base">
            Sync your Gmail account to start managing your emails efficiently
            with AI-powered insights and automation.
          </p>
        </div>

        {/* Error Message */}
        {error ? (
          <div
            className="text-sm text-red-500 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900 rounded-lg p-3"
            role="alert"
          >
            {error}
          </div>
        ) : null}

        {/* Action Button */}
        <div className="space-y-3 max-w-xs w-full mx-auto">
          <Button
            onClick={handleGmailConnect}
            disabled={loading}
            className="w-full"
          >
            {loading ? (
              <>
                <LoaderCircle className="h-5 w-5 mr-2 animate-spin" />
                Connecting...
              </>
            ) : (
              <>
                <Mail className="h-5 w-5 mr-2" />
                Sync Gmail Account
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}
