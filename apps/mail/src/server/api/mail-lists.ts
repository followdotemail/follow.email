import { BASE_URL } from "@/constants/base-url";

export interface PaginationDTO {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

export interface MailListResponse {
  emails: unknown[];
  pagination?: PaginationDTO;
}

export const mailListsFetcher = async (
  token: string,
  page: number = 1,
  limit: number = 20,
): Promise<MailListResponse> => {
  if (!BASE_URL) {
    throw new Error("BASE_URL is not configured");
  }

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
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }

  return response.json();
};
