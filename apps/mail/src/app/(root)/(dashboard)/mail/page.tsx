import { redirect } from "next/navigation";
import { auth } from "@clerk/nextjs/server";
import { checkUserConsentStatus } from "@/server/api/user-status";

export default async function Mail() {
  const { getToken } = await auth();
  const token = await getToken();

  if (!token) {
    return redirect("/sign-in");
  }

  const { isConnected } = await checkUserConsentStatus(token);

  return redirect(isConnected ? "/mail/inbox" : "/mail/onboarding");
}
