import { BASE_URL } from "@/constants/base-url";

export async function checkUserConsentStatus(token: string) {
  try {
    const response = await fetch(`${BASE_URL}/auth/status`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
    });

    const responseData = await response.json();

    console.log("User sync status response:", responseData);

    return {
      isConnected: responseData.gmail_consent,
    };
  } catch (error) {
    console.error("Error checking user sync status:", error);
    return {
      status: 500,
      isConnected: false,
      data: null,
      error: error instanceof Error ? error.message : "Unknown error",
    };
  }
}
