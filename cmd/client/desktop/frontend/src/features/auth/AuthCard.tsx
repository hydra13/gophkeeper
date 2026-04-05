import { Alert, Button, Card, Form, Input, Space, Typography } from "antd";

type Props = {
  mode: "login" | "register";
  busy: boolean;
  error: string | null;
  initialEmail?: string;
  onBack: () => void;
  onSubmit: (email: string, password: string) => Promise<void>;
};

type Values = {
  email: string;
  password: string;
};

export function AuthCard({
  mode,
  busy,
  error,
  initialEmail,
  onBack,
  onSubmit,
}: Props) {
  const [form] = Form.useForm<Values>();

  return (
    <Card className="auth-card" variant="borderless">
      <Space direction="vertical" size={18} style={{ width: "100%" }}>
        <div>
          <Typography.Title level={2} style={{ marginBottom: 8 }}>
            {mode === "login" ? "Login" : "Register"}
          </Typography.Title>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {mode === "login"
              ? "Open an existing session and continue with your synced secrets."
              : "Create a new account, then continue into the same desktop session automatically."}
          </Typography.Paragraph>
        </div>

        {error ? <Alert type="error" message={error} showIcon /> : null}

        <Form<Values>
          form={form}
          layout="vertical"
          initialValues={{ email: initialEmail ?? "", password: "" }}
          onFinish={(values) => onSubmit(values.email, values.password)}
        >
          <Form.Item
            label="Email"
            name="email"
            rules={[
              { required: true, message: "Enter your email" },
              { type: "email", message: "Enter a valid email" },
            ]}
          >
            <Input placeholder="you@example.com" size="large" />
          </Form.Item>

          <Form.Item
            label="Password"
            name="password"
            rules={[{ required: true, message: "Enter your password" }]}
          >
            <Input.Password placeholder="Password" size="large" />
          </Form.Item>

          <Space>
            <Button htmlType="submit" type="primary" size="large" loading={busy}>
              {mode === "login" ? "Login" : "Register"}
            </Button>
            <Button size="large" onClick={onBack}>
              Back
            </Button>
          </Space>
        </Form>
      </Space>
    </Card>
  );
}
