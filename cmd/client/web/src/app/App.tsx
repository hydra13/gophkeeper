import { useEffect, useMemo, useState } from "react";
import { App as AntApp, Alert, Layout, Modal, message } from "antd";
import { AuthCard } from "../features/auth/AuthCard";
import { StartScreen } from "../features/auth/StartScreen";
import { Workspace } from "../features/records/Workspace";
import { asMessage } from "../shared/lib/errors";
import { defaultSessionState } from "../shared/lib/records";
import { downloadBinary } from "../shared/api/uploads";
import { login, logout, register, restoreSession } from "../shared/api/auth";
import { createRecord, deleteRecord, getRecord, listRecords, updateRecord } from "../shared/api/records";
import { syncNow } from "../shared/api/sync";
import type {
  RecordDetails,
  RecordFilter,
  RecordListItem,
  RecordUpsertInput,
  SessionState,
} from "../shared/types";

type Screen = "loading" | "start" | "login" | "register" | "main";

export default function App() {
  const [messageApi, contextHolder] = message.useMessage();
  const [screen, setScreen] = useState<Screen>("loading");
  const [session, setSession] = useState<SessionState>(defaultSessionState());
  const [records, setRecords] = useState<RecordListItem[]>([]);
  const [selectedRecordId, setSelectedRecordId] = useState<number | null>(null);
  const [selectedRecord, setSelectedRecord] = useState<RecordDetails | null>(null);
  const [filter, setFilter] = useState<RecordFilter>("all");
  const [busy, setBusy] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const [registerSuccess, setRegisterSuccess] = useState<{
    open: boolean;
    email: string;
    password: string;
  }>({ open: false, email: "", password: "" });
  const [prefillEmail, setPrefillEmail] = useState("");

  useEffect(() => {
    void bootstrap();
  }, []);

  useEffect(() => {
    if (screen !== "main") {
      return;
    }
    void refreshRecords(filter);
  }, [filter, screen]);

  useEffect(() => {
    if (selectedRecordId == null || screen !== "main") {
      setSelectedRecord(null);
      return;
    }
    void loadRecord(selectedRecordId);
  }, [selectedRecordId, screen]);

  const selectedListRecord = useMemo(
    () => records.find((record) => record.id === selectedRecordId) ?? null,
    [records, selectedRecordId],
  );

  async function bootstrap() {
    setBusy(true);
    try {
      const state = await restoreSession();
      setSession(state);
      setScreen(state.authenticated ? "main" : "start");
    } catch (error) {
      messageApi.error(asMessage(error));
      setScreen("start");
    } finally {
      setBusy(false);
    }
  }

  async function refreshRecords(nextFilter = filter) {
    setBusy(true);
    try {
      const nextRecords = await listRecords(nextFilter);
      setRecords(nextRecords);
      setSelectedRecordId((currentId) => {
        if (nextRecords.length === 0) {
          return null;
        }
        if (currentId && nextRecords.some((record) => record.id === currentId)) {
          return currentId;
        }
        return nextRecords[0].id;
      });
    } catch (error) {
      messageApi.error(asMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function loadRecord(recordId: number) {
    try {
      const record = await getRecord(recordId);
      setSelectedRecord(record);
    } catch (error) {
      messageApi.error(asMessage(error));
    }
  }

  async function handleLogin(email: string, password: string) {
    setBusy(true);
    setAuthError(null);
    try {
      const nextSession = await login(email, password);
      setSession(nextSession);
      setScreen("main");
      await refreshRecords(filter);
    } catch (error) {
      setAuthError(asMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function handleRegister(email: string, password: string) {
    setBusy(true);
    setAuthError(null);
    try {
      await register(email, password);
      setRegisterSuccess({
        open: true,
        email,
        password,
      });
    } catch (error) {
      setAuthError(asMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function handleRegisterSuccessOk() {
    setBusy(true);
    try {
      const nextSession = await login(
        registerSuccess.email,
        registerSuccess.password,
      );
      setSession(nextSession);
      setRegisterSuccess({ open: false, email: "", password: "" });
      setScreen("main");
      await refreshRecords(filter);
    } catch (error) {
      setRegisterSuccess({ open: false, email: "", password: "" });
      setPrefillEmail(registerSuccess.email);
      setScreen("login");
      setAuthError(asMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function handleLogout() {
    setBusy(true);
    try {
      const nextSession = await logout();
      setSession(nextSession);
      setRecords([]);
      setSelectedRecordId(null);
      setSelectedRecord(null);
      setScreen("start");
      messageApi.success("Logged out");
    } catch (error) {
      messageApi.error(asMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function handleCreateRecord(input: RecordUpsertInput) {
    setBusy(true);
    try {
      const created = await createRecord(input);
      messageApi.success("Record created");
      await refreshRecords(filter);
      setSelectedRecordId(created.id);
    } catch (error) {
      throw new Error(asMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function handleUpdateRecord(input: RecordUpsertInput) {
    setBusy(true);
    try {
      const updated = await updateRecord(input);
      messageApi.success("Record updated");
      await refreshRecords(filter);
      setSelectedRecordId(updated.id);
    } catch (error) {
      throw new Error(asMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function handleDeleteRecord(recordId: number) {
    setBusy(true);
    try {
      await deleteRecord(recordId);
      messageApi.success("Record deleted");
      await refreshRecords(filter);
    } catch (error) {
      messageApi.error(asMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function handleSync() {
    setBusy(true);
    try {
      const result = await syncNow();
      messageApi.success(result.message);
      await refreshRecords(filter);
    } catch (error) {
      messageApi.error(asMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function handleDownloadBinary(record: RecordDetails) {
    try {
      await downloadBinary(record.id, record.name);
      messageApi.success(`Downloaded ${record.name}`);
    } catch (error) {
      messageApi.error(asMessage(error));
    }
  }

  return (
    <AntApp>
      {contextHolder}
      <Layout className="web-shell">
        {screen === "loading" ? (
          <div className="center-stage">
            <div className="hero-card muted">Restoring session...</div>
          </div>
        ) : null}

        {screen === "start" ? (
          <StartScreen
            session={session}
            onLogin={() => {
              setAuthError(null);
              setScreen("login");
            }}
            onRegister={() => {
              setAuthError(null);
              setScreen("register");
            }}
          />
        ) : null}

        {screen === "login" ? (
          <div className="center-stage">
            <AuthCard
              mode="login"
              busy={busy}
              error={authError}
              initialEmail={prefillEmail}
              onBack={() => {
                setAuthError(null);
                setScreen("start");
              }}
              onSubmit={handleLogin}
            />
          </div>
        ) : null}

        {screen === "register" ? (
          <div className="center-stage">
            <AuthCard
              mode="register"
              busy={busy}
              error={authError}
              onBack={() => {
                setAuthError(null);
                setScreen("start");
              }}
              onSubmit={handleRegister}
            />
          </div>
        ) : null}

        {screen === "main" ? (
          <Workspace
            busy={busy}
            session={session}
            filter={filter}
            records={records}
            selectedRecord={selectedRecord}
            selectedListRecord={selectedListRecord}
            onFilterChange={setFilter}
            onRefresh={() => refreshRecords(filter)}
            onSelectRecord={(recordId) => setSelectedRecordId(recordId)}
            onCreateRecord={handleCreateRecord}
            onUpdateRecord={handleUpdateRecord}
            onDeleteRecord={handleDeleteRecord}
            onSync={handleSync}
            onLogout={handleLogout}
            onDownloadBinary={handleDownloadBinary}
          />
        ) : null}

        <Modal
          open={registerSuccess.open}
          title="registered successfully"
          okText="OK"
          cancelButtonProps={{ style: { display: "none" } }}
          closable={false}
          maskClosable={false}
          onOk={() => void handleRegisterSuccessOk()}
        >
          <p>Your account was created. Press OK to login automatically.</p>
        </Modal>

        {authError && screen === "start" ? (
          <div className="floating-alert">
            <Alert type="error" message={authError} showIcon />
          </div>
        ) : null}
      </Layout>
    </AntApp>
  );
}
