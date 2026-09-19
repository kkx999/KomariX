import React from "react";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";
import { Button, Dialog, Flex } from "@radix-ui/themes";
import { Activity, Eraser, Trash2 } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import NumberPicker from "@/components/ui/number-picker";
import Loading from "@/components/loading";
import { toast } from "sonner";

interface Log {
  id: number;
  ip: string;
  uuid: string;
  message: string;
  msg_type: string;
  time: string;
}
const LogPage = () => {
  const [loading, setLoading] = React.useState<boolean>(true);
  const [logs, setLogs] = React.useState<Log[]>([]);
  const [error, setError] = React.useState<string | null>(null);
  const [page, setPage] = React.useState<number>(1);
  const [total, setTotal] = React.useState<number>(1);
  const [limit, setLimit] = React.useState<number>(10);
  const [actionLoading, setActionLoading] = React.useState(false);
  const [clearDialogOpen, setClearDialogOpen] = React.useState(false);
  const [t] = useTranslation();
  const navigate = useNavigate();

  const fetchLogs = React.useCallback(
    async (requestedPage = page) => {
      setLoading(true);
      setError(null);
      try {
        const response = await fetch(
          `/api/admin/logs?limit=${limit}&page=${requestedPage}`,
        );
        if (!response.ok) {
          throw new Error("Failed to fetch logs");
        }
        const data = await response.json();
        setLogs(data.data.logs);
        setTotal(data.data.total);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        setLoading(false);
      }
    },
    [limit, page],
  );

  React.useEffect(() => {
    void fetchLogs();
  }, [fetchLogs]);

  const runLogCleanup = async (clearAll = false) => {
    setActionLoading(true);
    try {
      const response = await fetch(
        clearAll ? "/api/admin/logs/clear" : "/api/admin/logs/cleanup",
        { method: "POST" },
      );
      const data = await response.json();
      if (!response.ok || data.status === "error") {
        throw new Error(data.message || "Log cleanup failed");
      }
      const deleted = Number(data.data?.deleted || 0);
      toast.success(
        clearAll
          ? t("logs.clear_success", { count: deleted })
          : t("logs.cleanup_success", { count: deleted }),
      );
      setPage(1);
      await fetchLogs(1);
      if (clearAll) setClearDialogOpen(false);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("logs.cleanup_failed"),
      );
    } finally {
      setActionLoading(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / limit));
  // 计算分页页码，显示当前页及前后1页，两端省略号
  const siblingsCount = 1;
  let pageNumbers: (number | string)[] = [];
  const leftSibling = Math.max(page - siblingsCount, 1);
  const rightSibling = Math.min(page + siblingsCount, totalPages);
  const showLeftDots = leftSibling > 2;
  const showRightDots = rightSibling < totalPages - 1;
  // 始终包含第一页
  pageNumbers.push(1);
  // 左侧省略或中间连续页
  if (showLeftDots) {
    pageNumbers.push("...");
  } else {
    for (let i = 2; i < leftSibling; i++) pageNumbers.push(i);
  }
  // 中间页，仅当不重复首尾页时加入
  for (let i = leftSibling; i <= rightSibling; i++) {
    if (i > 1 && i < totalPages) pageNumbers.push(i);
  }
  // 右侧省略或中间连续页
  if (showRightDots) {
    pageNumbers.push("...");
  } else {
    for (let i = rightSibling + 1; i < totalPages; i++) pageNumbers.push(i);
  }
  // 始终包含最后一页（如果大于1）
  if (totalPages > 1) pageNumbers.push(totalPages);

  if (loading) {
    return <Loading />;
  }
  if (error) {
    return <div>Error: {error}</div>;
  }

  return (
    <div className="km-page-admin-log flex flex-col gap-2 p-4">
      <div className="km-log-toolbar flex flex-wrap justify-between items-center gap-2">
        <h1 className="text-2xl font-bold">{t("logs.title")}</h1>
        <div className="flex flex-wrap items-center justify-end gap-2">
          <Button variant="soft" onClick={() => navigate("/admin/pprof")}>
            <Activity size={16} />
            {t("pprof.title")}
          </Button>
          <Button
            variant="soft"
            disabled={actionLoading}
            onClick={() => void runLogCleanup(false)}
          >
            <Eraser size={16} />
            {t("logs.cleanup")}
          </Button>
          <Dialog.Root open={clearDialogOpen} onOpenChange={setClearDialogOpen}>
            <Dialog.Trigger>
              <Button variant="soft" color="red" disabled={actionLoading}>
                <Trash2 size={16} />
                {t("logs.clear_all")}
              </Button>
            </Dialog.Trigger>
            <Dialog.Content maxWidth="420px">
              <Dialog.Title>{t("logs.clear_all")}</Dialog.Title>
              <Dialog.Description>
                {t("logs.clear_confirm")}
              </Dialog.Description>
              <Flex gap="3" mt="4" justify="end">
                <Dialog.Close>
                  <Button variant="soft" color="gray">
                    {t("common.cancel")}
                  </Button>
                </Dialog.Close>
                <Button
                  color="red"
                  disabled={actionLoading}
                  onClick={() => void runLogCleanup(true)}
                >
                  <Trash2 size={16} />
                  {t("logs.clear_all")}
                </Button>
              </Flex>
            </Dialog.Content>
          </Dialog.Root>
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">
              {t("logs.page_size")}
            </span>
            <NumberPicker
              defaultValue={limit}
              onChange={(value) => {
                setLimit(value);
                setPage(1);
              }}
              min={1}
              max={100}
            />
          </div>
        </div>
      </div>
      <div className="km-log-output rounded-lg overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>IP</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Message</TableHead>
              <TableHead>Time</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {logs.map((log) => (
              <TableRow key={log.id}>
                <TableCell>
                  <Dialog.Root>
                    <Dialog.Trigger>
                      <label className="hover:underline font-bold">
                        {log.id}
                      </label>
                    </Dialog.Trigger>
                    <Dialog.Content>
                      <Dialog.Title>{t("log.title")}</Dialog.Title>
                      <Flex direction="column" gap="1">
                        <label className="font-bold">ID</label>
                        <label className="text-sm">{log.id}</label>
                        <label className="font-bold">IP</label>
                        <label className="text-sm">{log.ip}</label>
                        <label className="font-bold">UUID</label>
                        <label className="text-sm">{log.uuid}</label>
                        <label className="font-bold">Type</label>
                        <label className="text-sm">{log.msg_type}</label>
                        <label className="font-bold">Message</label>
                        <label className="text-sm">{log.message}</label>
                        <label className="font-bold">Time</label>
                        <label className="text-sm">
                          {new Date(log.time).toLocaleString()}
                        </label>
                      </Flex>
                      <Flex justify={"end"}>
                        <Dialog.Close>
                          <Button variant="soft">{t("common.close")}</Button>
                        </Dialog.Close>
                      </Flex>
                    </Dialog.Content>
                  </Dialog.Root>
                </TableCell>
                <TableCell>{log.ip}</TableCell>
                <TableCell>{log.msg_type}</TableCell>
                <TableCell>
                  {log.message.length > 75
                    ? `${log.message.slice(0, 75)}...`
                    : log.message}
                </TableCell>
                <TableCell>{new Date(log.time).toLocaleString()}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {/* 分页数字按钮 */}
      <div className="flex justify-center items-center space-x-2 mt-4 gap-2">
        <Button
          disabled={page === 1}
          onClick={() => setPage((p) => Math.max(1, p - 1))}
          title={t("common.previous_page", "Previous page")}
          aria-label={t("common.previous_page", "Previous page")}
        >
          {"<"}
        </Button>
        {pageNumbers.map((p, i) =>
          typeof p === "number" ? (
            <Button
              key={i}
              variant={p === page ? "solid" : "soft"}
              onClick={() => setPage(p)}
            >
              {p}
            </Button>
          ) : (
            <span key={i} className="px-2">
              ...
            </span>
          )
        )}
        <Button
          disabled={page === totalPages}
          onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
          title={t("common.next_page", "Next page")}
          aria-label={t("common.next_page", "Next page")}
        >
          {">"}
        </Button>
      </div>
    </div>
  );
};
export default LogPage;
