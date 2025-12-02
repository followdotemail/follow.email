import { BASE_URL } from "@/constants/base-url";

interface PaginationDTO {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

interface MailListResponse {
  emails: unknown[];
  pagination?: PaginationDTO;
}

export async function fetchMailLists(
  token: string,
  page: number = 1,
  limit: number = 20,
) {
  try {
    const url = new URL(`${BASE_URL}/emails`);
    url.searchParams.set("page", String(page));
    url.searchParams.set("limit", String(limit));

    const response = await fetch(url.toString(), {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Server error: ${response.status}`);
    }

    const responseData = (await response.json()) as MailListResponse;

    return {
      status: response.status,
      data: responseData.emails || [],
      pagination: responseData.pagination,
      success: true,
    };
  } catch (error) {
    console.error("Error fetching mail lists:", error);
    return {
      status: 500,
      data: [],
      pagination: undefined,
      success: false,
      error: error instanceof Error ? error.message : "Unknown error",
    };
  }
}
