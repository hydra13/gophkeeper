import { EyeInvisibleOutlined, EyeOutlined } from "@ant-design/icons";
import { Button, Card, Descriptions, Empty, Space, Tag, Typography } from "antd";
import { useState } from "react";
import type { RecordDetails } from "../../shared/types";

type Props = {
  record: RecordDetails | null;
  onSaveBinary: (recordId: number) => Promise<void>;
};

function SecretField({ value }: { value: string }) {
  const [visible, setVisible] = useState(false);
  return (
    <Space>
      <Typography.Text copyable={visible ? { text: value } : false}>
        {visible ? value : "••••••••"}
      </Typography.Text>
      <Button
        type="text"
        size="small"
        icon={visible ? <EyeInvisibleOutlined /> : <EyeOutlined />}
        onClick={() => setVisible(!visible)}
      />
    </Space>
  );
}

export function RecordDetailsPane({ record, onSaveBinary }: Props) {
  if (!record) {
    return (
      <Card className="panel-card">
        <Empty description="Select a record to inspect it" />
      </Card>
    );
  }

  return (
    <Card
      className="panel-card"
      title={
        <Space>
          <Typography.Text strong>{record.name}</Typography.Text>
          <Tag>{record.type}</Tag>
        </Space>
      }
    >
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="ID">{record.id}</Descriptions.Item>
        <Descriptions.Item label="Revision">{record.revision}</Descriptions.Item>
        <Descriptions.Item label="Metadata">
          {record.metadata || <Typography.Text type="secondary">None</Typography.Text>}
        </Descriptions.Item>
        <Descriptions.Item label="Device">{record.deviceId || "unknown"}</Descriptions.Item>
        <Descriptions.Item label="Created">{record.createdAt || "unknown"}</Descriptions.Item>
        <Descriptions.Item label="Updated">{record.updatedAt || "unknown"}</Descriptions.Item>

        {record.type === "login" ? (
          <>
            <Descriptions.Item label="Login">{record.payload.login}</Descriptions.Item>
            <Descriptions.Item label="Password">
              <SecretField value={record.payload.password} />
            </Descriptions.Item>
          </>
        ) : null}

        {record.type === "text" ? (
          <Descriptions.Item label="Content">
            <Typography.Paragraph style={{ whiteSpace: "pre-wrap", marginBottom: 0 }}>
              {record.payload.content}
            </Typography.Paragraph>
          </Descriptions.Item>
        ) : null}

        {record.type === "card" ? (
          <>
            <Descriptions.Item label="Number">
              <SecretField value={record.payload.number} />
            </Descriptions.Item>
            <Descriptions.Item label="Holder">{record.payload.holder}</Descriptions.Item>
            <Descriptions.Item label="Expiry">{record.payload.expiry}</Descriptions.Item>
            <Descriptions.Item label="CVV">
              <SecretField value={record.payload.cvv} />
            </Descriptions.Item>
          </>
        ) : null}

        {record.type === "binary" ? (
          <>
            <Descriptions.Item label="Payload version">
              {record.payloadVersion}
            </Descriptions.Item>
            <Descriptions.Item label="Actions">
              <Button onClick={() => void onSaveBinary(record.id)}>Save file...</Button>
            </Descriptions.Item>
          </>
        ) : null}
      </Descriptions>
    </Card>
  );
}
