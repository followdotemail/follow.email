"use client";

import { SignIn, useAuth } from "@clerk/nextjs";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

export default function Page() {
  console.log("🎯 Page component loaded!");

  const { getToken, isSignedIn, isLoaded } = useAuth();
  const router = useRouter();
  const [isRegistering, setIsRegistering] = useState(false);
  const [error, setError] = useState<string | null>(null);

  console.log("🔍 Auth state:", { isLoaded, isSignedIn, isRegistering });

  // Check if Clerk is properly configured
  useEffect(() => {
    if (!process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY) {
      console.error("❌ NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY is not set!");
      setError(
        "Clerk authentication is not properly configured. Please check your environment variables."
      );
      return;
    }
  }, []);

  useEffect(() => {
    console.log("🔄 useEffect triggered:", {
      isLoaded,
      isSignedIn,
      isRegistering,
    });

    const handleUserRegistration = async () => {
      console.log("🚀 handleUserRegistration called");
      if (isLoaded && isSignedIn && !isRegistering) {
        console.log("✅ Starting registration process...");
        setIsRegistering(true);
        setError(null);

        try {
          const token = await getToken();
          if (!token) {
            setError("Failed to get authentication token");
            return;
          }
        } catch (err) {
          console.error("Registration error:", err);
          setError("An unexpected error occurred during registration");
        } finally {
          setIsRegistering(false);
        }
      }
    };

    handleUserRegistration();
  }, [isLoaded, isSignedIn, getToken, router, isRegistering]);

  // Show loading state during registration
  if (isSignedIn && isRegistering) {
    return (
      <main className="flex min-h-svh items-center justify-center dark">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-white mx-auto mb-4"></div>
          <p className="text-white">Completing registration...</p>
        </div>
      </main>
    );
  }

  // Show error state if registration failed
  if (error) {
    return (
      <main className="flex min-h-svh items-center justify-center dark">
        <div className="text-center max-w-md">
          <div className="bg-red-900/20 border border-red-500 rounded-lg p-4 mb-4">
            <p className="text-red-400 font-medium">Registration Failed</p>
            <p className="text-red-300 text-sm mt-2">{error}</p>
          </div>
          <button
            onClick={() => {
              setError(null);
              window.location.reload();
            }}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md"
          >
            Try Again
          </button>
        </div>
      </main>
    );
  }

  return (
    <main className="flex min-h-svh items-center justify-center dark">
      <SignIn  routing="path" path="/sign-in" withSignUp />
    </main>
  );
}
