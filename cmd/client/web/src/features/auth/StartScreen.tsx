import { Button, Card, Space, Tag, Typography } from "antd";
import type { SessionState } from "../../shared/types";

type Props = {
  session: SessionState;
  onLogin: () => void;
  onRegister: () => void;
};

export function StartScreen({ session, onLogin, onRegister }: Props) {
  return (
    <div className="center-stage">
      <Card className="hero-card" variant="borderless">
        <Space direction="vertical" size={20}>
          <div>
            <Tag color="blue">{session.appName}</Tag>
            <Typography.Title level={1} className="hero-title">
              Secure web vault
            </Typography.Title>
            <Typography.Paragraph className="hero-copy">
              Work with the same GophKeeper server from the browser and manage
              login, text, binary, and card records from one web workspace.
            </Typography.Paragraph>
          </div>

          <Space size={12}>
            <Button type="primary" size="large" onClick={onLogin}>
              Login
            </Button>
            <Button size="large" onClick={onRegister}>
              Register
            </Button>
          </Space>

          <Space wrap>
            <Tag>Server: {session.serverAddress}</Tag>
            <Tag>Device: {session.deviceId || "web-client"}</Tag>
            <Tag>Version: {session.version}</Tag>
          </Space>
        </Space>
      </Card>
    </div>
  );
}
