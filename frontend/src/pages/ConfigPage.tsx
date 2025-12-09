import { ConfigEditor } from '@/components/config/ConfigEditor';

const CONFIG_KEYS = [
  {
    key: 'library.root_path',
    label: 'Library Root Path',
    description: 'The root directory for your media library',
  },
  {
    key: 'download.tmp_path',
    label: 'Download Temporary Path',
    description: 'Temporary directory for downloads and processing',
  },
  {
    key: 'metadata.provider',
    label: 'Metadata Provider',
    description: 'Primary metadata provider configuration',
  },
  {
    key: 'server.host',
    label: 'Server Host',
    description: 'Server bind address',
  },
];

export function ConfigPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Configuration</h1>
        <p className="text-muted-foreground">
          Manage system configuration settings
        </p>
      </div>

      <div className="space-y-4">
        {CONFIG_KEYS.map((config) => (
          <ConfigEditor
            key={config.key}
            configKey={config.key}
            label={config.label}
            description={config.description}
          />
        ))}
      </div>
    </div>
  );
}
