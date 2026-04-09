import { useEffect } from "react";
import {
  Button,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Typography,
} from "antd";
import type { RecordDetails, RecordType, RecordUpsertInput } from "../../shared/types";

type Props = {
  busy: boolean;
  open: boolean;
  mode: "create" | "update";
  initialRecord?: RecordDetails | null;
  onClose: () => void;
  onPickFile: () => Promise<string>;
  onSubmit: (input: RecordUpsertInput) => Promise<void>;
};

type FormValues = RecordUpsertInput;

const typeOptions: Array<{ label: string; value: RecordType }> = [
  { label: "Login", value: "login" },
  { label: "Text", value: "text" },
  { label: "Binary", value: "binary" },
  { label: "Card", value: "card" },
];

export function RecordFormModal({
  busy,
  open,
  mode,
  initialRecord,
  onClose,
  onPickFile,
  onSubmit,
}: Props) {
  const [form] = Form.useForm<FormValues>();
  const recordType = Form.useWatch("type", form) as RecordType | undefined;

  useEffect(() => {
    if (!open) {
      return;
    }

    form.setFieldsValue(toFormValues(initialRecord, mode));
  }, [form, initialRecord, mode, open]);

  return (
    <Modal
      destroyOnHidden
      open={open}
      title={mode === "create" ? "Add record" : "Update record"}
      width={720}
      okText={mode === "create" ? "Create" : "Save"}
      confirmLoading={busy}
      onCancel={onClose}
      onOk={() => form.submit()}
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        onFinish={async (values) => {
          await onSubmit(values);
          onClose();
        }}
      >
        <Form.Item
          label="Type"
          name="type"
          rules={[{ required: true, message: "Select record type" }]}
        >
          <Select disabled={mode === "update"} options={typeOptions} />
        </Form.Item>

        <Form.Item
          label="Name"
          name="name"
          rules={[{ required: true, message: "Enter record name" }]}
        >
          <Input />
        </Form.Item>

        <Form.Item label="Metadata" name="metadata">
          <Input.TextArea rows={3} />
        </Form.Item>

        {recordType === "login" ? (
          <>
            <Form.Item label="Login" name="login" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item
              label="Password"
              name="password"
              rules={[{ required: true }]}
            >
              <Input.Password />
            </Form.Item>
          </>
        ) : null}

        {recordType === "text" ? (
          <Form.Item label="Content" name="content" rules={[{ required: true }]}>
            <Input.TextArea rows={6} />
          </Form.Item>
        ) : null}

        {recordType === "card" ? (
          <>
            <Form.Item label="Number" name="number" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item label="Holder" name="holder" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item label="Expiry" name="expiry" rules={[{ required: true }]}>
              <Input placeholder="MM/YY" />
            </Form.Item>
            <Form.Item label="CVV" name="cvv" rules={[{ required: true }]}>
              <Input.Password />
            </Form.Item>
          </>
        ) : null}

        {recordType === "binary" ? (
          <>
            <Form.Item
              label={mode === "create" ? "File path" : "New file path"}
              name="filePath"
              rules={
                mode === "create"
                  ? [{ required: true, message: "Select file for upload" }]
                  : undefined
              }
            >
              <Input
                addonAfter={
                  <Button
                    type="link"
                    onClick={async () => {
                      const path = await onPickFile();
                      if (path) {
                        form.setFieldValue("filePath", path);
                      }
                    }}
                  >
                    Browse
                  </Button>
                }
              />
            </Form.Item>
            <Typography.Paragraph type="secondary" style={{ marginTop: -8 }}>
              {mode === "create"
                ? "Binary content is uploaded after the record is created."
                : "Leave the file path empty to keep the current binary content."}
            </Typography.Paragraph>
          </>
        ) : null}
      </Form>
    </Modal>
  );
}

function toFormValues(
  record: RecordDetails | null | undefined,
  mode: "create" | "update",
): FormValues {
  if (!record || mode === "create") {
    return {
      id: 0,
      type: "login",
      name: "",
      metadata: "",
      login: "",
      password: "",
      content: "",
      number: "",
      holder: "",
      expiry: "",
      cvv: "",
      filePath: "",
    };
  }

  return {
    id: record.id,
    type: record.type,
    name: record.name,
    metadata: record.metadata,
    login: record.payload.login,
    password: record.payload.password,
    content: record.payload.content,
    number: record.payload.number,
    holder: record.payload.holder,
    expiry: record.payload.expiry,
    cvv: record.payload.cvv,
    filePath: "",
  };
}
