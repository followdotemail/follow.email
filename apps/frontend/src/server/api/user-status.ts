import { BASE_URL } from "@/constants/base-url";

export async function checkUserSyncStatus(token: string) {
  try {
    const response = await fetch(
      `${BASE_URL}/emails/sync/status?provider=gmail`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      }
    );

    return {
      status: response.status,
      isConnected: response.status === 200,
      data: response.status === 200 ? await response.json() : null,
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
