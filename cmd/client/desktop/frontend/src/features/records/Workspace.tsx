import { useState } from "react";
import {
  Button,
  Card,
  Layout,
  Popconfirm,
  Segmented,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import {
  DeleteOutlined,
  DownloadOutlined,
  LogoutOutlined,
  PlusOutlined,
  ReloadOutlined,
  SyncOutlined,
  EditOutlined,
} from "@ant-design/icons";
import type {
  RecordDetails,
  RecordFilter,
  RecordListItem,
  RecordUpsertInput,
  SessionState,
} from "../../shared/types";
import { RecordDetailsPane } from "./RecordDetailsPane";
import { RecordFormModal } from "./RecordFormModal";

type Props = {
  busy: boolean;
  session: SessionState;
  filter: RecordFilter;
  records: RecordListItem[];
  selectedRecord: RecordDetails | null;
  selectedListRecord: RecordListItem | null;
  onFilterChange: (filter: RecordFilter) => void;
  onRefresh: () => Promise<void>;
  onSelectRecord: (recordId: number) => void;
  onCreateRecord: (input: RecordUpsertInput) => Promise<void>;
  onUpdateRecord: (input: RecordUpsertInput) => Promise<void>;
  onDeleteRecord: (recordId: number) => Promise<void>;
  onSync: () => Promise<void>;
  onLogout: () => Promise<void>;
  onPickFile: () => Promise<string>;
  onSaveBinary: (recordId: number) => Promise<void>;
  onDownloadBinary: (recordId: number, savePath: string) => Promise<void>;
};

export function Workspace({
  busy,
  session,
  filter,
  records,
  selectedRecord,
  selectedListRecord,
  onFilterChange,
  onRefresh,
  onSelectRecord,
  onCreateRecord,
  onUpdateRecord,
  onDeleteRecord,
  onSync,
  onLogout,
  onPickFile,
  onSaveBinary,
}: Props) {
  const [modalState, setModalState] = useState<{
    open: boolean;
    mode: "create" | "update";
  }>({ open: false, mode: "create" });

  return (
    <Layout className="workspace-layout">
      <Layout.Header className="workspace-header">
        <div>
          <Typography.Title level={3} style={{ color: "#fff", margin: 0 }}>
            GophKeeper Desktop
          </Typography.Title>
          <Typography.Text style={{ color: "rgba(255,255,255,0.7)" }}>
            {session.email || "anonymous"} on {session.serverAddress}
          </Typography.Text>
        </div>

        <Space>
          <Button icon={<SyncOutlined />} onClick={() => void onSync()} loading={busy}>
            Sync
          </Button>
          <Button
            icon={<LogoutOutlined />}
            onClick={() => void onLogout()}
            danger
          >
            Logout
          </Button>
        </Space>
      </Layout.Header>

      <Layout.Content className="workspace-content">
        <div className="toolbar-row">
          <Space wrap>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setModalState({ open: true, mode: "create" })}
            >
              Add
            </Button>
            <Button
              icon={<EditOutlined />}
              disabled={!selectedRecord}
              onClick={() => setModalState({ open: true, mode: "update" })}
            >
              Update
            </Button>
            <Popconfirm
              title="Delete record?"
              description="This action removes the selected record."
              disabled={!selectedListRecord}
              onConfirm={() =>
                selectedListRecord
                  ? onDeleteRecord(selectedListRecord.id)
                  : Promise.resolve()
              }
            >
              <Button
                icon={<DeleteOutlined />}
                danger
                disabled={!selectedListRecord}
              >
                Delete
              </Button>
            </Popconfirm>
            <Button
              icon={<DownloadOutlined />}
              disabled={selectedRecord?.type !== "binary"}
              onClick={() =>
                selectedRecord ? onSaveBinary(selectedRecord.id) : Promise.resolve()
              }
            >
              Save file
            </Button>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => void onRefresh()}
              loading={busy}
            >
              Refresh
            </Button>
          </Space>

          <Segmented<RecordFilter>
            value={filter}
            options={[
              { label: "All", value: "all" },
              { label: "Login", value: "login" },
              { label: "Text", value: "text" },
              { label: "Binary", value: "binary" },
              { label: "Card", value: "card" },
            ]}
            onChange={(value) => onFilterChange(value)}
          />
        </div>

        <div className="workspace-grid">
          <Card className="panel-card" title="Records">
            <Table<RecordListItem>
              rowKey="id"
              loading={busy}
              pagination={false}
              size="small"
              dataSource={records}
              rowClassName={(record) =>
                record.id === selectedListRecord?.id ? "selected-row" : ""
              }
              onRow={(record) => ({
                onClick: () => onSelectRecord(record.id),
              })}
              columns={[
                { title: "ID", dataIndex: "id", width: 90 },
                {
                  title: "Type",
                  dataIndex: "type",
                  width: 110,
                  render: (value: string) => <Tag>{value}</Tag>,
                },
                { title: "Name", dataIndex: "name" },
                { title: "Revision", dataIndex: "revision", width: 110 },
                {
                  title: "Metadata",
                  dataIndex: "metadataPreview",
                  render: (value: string) => value || "—",
                },
              ]}
            />
          </Card>

          <RecordDetailsPane
            record={selectedRecord}
            onSaveBinary={onSaveBinary}
          />
        </div>
      </Layout.Content>

      <RecordFormModal
        busy={busy}
        open={modalState.open}
        mode={modalState.mode}
        initialRecord={modalState.mode === "update" ? selectedRecord : null}
        onClose={() => setModalState({ open: false, mode: "create" })}
        onPickFile={onPickFile}
        onSubmit={(input) =>
          modalState.mode === "create"
            ? onCreateRecord(input)
            : onUpdateRecord(input)
        }
      />
    </Layout>
  );
}
