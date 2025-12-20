import { BASE_URL } from "@/constants/base-url";

export async function checkUserConsentStatus(token: string) {
  try {
    if (!BASE_URL) {
      throw new Error("BASE_URL is not configured");
    }

    const response = await fetch(`${BASE_URL}/auth/status`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const responseData = await response.json();

    return {
      isConnected: Boolean(responseData.gmail_consent),
    };
  } catch (error) {
    console.error("Error checking user consent status:", error);
    return {
      isConnected: false,
    };
  }
}
