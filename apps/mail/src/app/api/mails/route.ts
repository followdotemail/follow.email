import { NextRequest, NextResponse } from "next/server";
import { auth } from "@clerk/nextjs/server";
import { fetchMailLists } from "@/server/api/mail-lists";

export async function GET(req: NextRequest) {
  try {
    const { getToken } = await auth();

    if (!getToken) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    }

    const token = await getToken();

    if (!token) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    }

    const { searchParams } = new URL(req.url);
    const page = Number(searchParams.get("page") ?? "1");
    const limit = Number(searchParams.get("limit") ?? "20");

    const result = await fetchMailLists(token, page, limit);

    return NextResponse.json(result, {
      status: result.status ?? 200,
    });
  } catch (error) {
    console.error("Error in /api/mails route:", error);
    return NextResponse.json(
      {
        status: 500,
        data: [],
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
      },
      { status: 500 },
    );
  }
}


