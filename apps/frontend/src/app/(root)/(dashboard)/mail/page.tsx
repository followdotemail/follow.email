import { redirect } from "next/navigation";
import { auth } from "@clerk/nextjs/server";
import { checkUserSyncStatus } from "@/server/api/user-status";

export default async function Mail() {
  const { getToken } = await auth();
  const token = await getToken();

  if (!token) {
    return redirect("/sign-in");
  }

  const { isConnected } = await checkUserSyncStatus(token);

  return redirect(isConnected ? "/mail/inbox" : "/mail/onboarding");
}
