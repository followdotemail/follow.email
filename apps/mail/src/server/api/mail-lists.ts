import { BASE_URL } from "@/constants/base-url";

export async function fetchMailLists(token: string) {
  try {
    const response = await fetch(`${BASE_URL}/emails`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Server error: ${response.status}`);
    }

    const responseData = await response.json();
    // console.log("Mail lists response:", responseData);

    return {
      status: response.status,
      data: responseData.emails || [],
      success: true,
    };
  } catch (error) {
    console.error("Error fetching mail lists:", error);
    return {
      status: 500,
      data: [],
      success: false,
      error: error instanceof Error ? error.message : "Unknown error",
    };
  }
}
