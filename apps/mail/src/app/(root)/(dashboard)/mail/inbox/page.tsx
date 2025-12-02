import { cookies } from "next/headers";
import { accounts } from "@/constants/mail-data";
import { Mail, type EmailData } from "@/components/mail/mail";
import { fetchMailLists } from "@/server/api/mail-lists";
import { auth } from "@clerk/nextjs/server";
import { redirect } from "next/navigation";

export default async function MailPage() {
  const { getToken } = await auth();

  if (!getToken) {
    redirect("/sign-in");
  }

  const token = await getToken();
  if (!token) {
    redirect("/sign-in");
  }

  const mailData = await fetchMailLists(token, 1, 20);

  const cookieStore = await cookies();
  const layout = cookieStore.get("react-resizable-panels:layout:mail");
  const collapsed = cookieStore.get("react-resizable-panels:collapsed");

  const defaultLayout = layout ? JSON.parse(layout.value) : undefined;
  const defaultCollapsed = collapsed ? JSON.parse(collapsed.value) : undefined;

  const mails = (mailData.data || []) as EmailData[];

  return (
    <>
      <div className=" flex-col md:flex">
        <Mail
          accounts={accounts}
          mails={mails}
          initialPagination={mailData.pagination}
          defaultLayout={defaultLayout}
          defaultCollapsed={defaultCollapsed}
          navCollapsedSize={4}
        />
      </div>
    </>
  );
}
