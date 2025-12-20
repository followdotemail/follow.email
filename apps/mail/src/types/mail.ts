export interface EmailData {
  id: string;
  clerk_id: string;
  message_id: string;
  thread_id: string;
  subject: string;
  from_email: string;
  from_name: string;
  to_emails: string;
  cc_emails: string;
  bcc_emails: string;
  updated_at: string;
  is_read: boolean;
  is_important: boolean;
  has_attachments: boolean;
  labels: string;
  last_sync_at: string;
}

export interface Pagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

export interface MailProps {
  accounts: {
    label: string;
    email: string;
    icon: React.ReactNode;
  }[];
  mails: EmailData[];
  initialPagination?: Pagination;
  defaultLayout: number[] | undefined;
  defaultCollapsed?: boolean;
  navCollapsedSize: number;
}
