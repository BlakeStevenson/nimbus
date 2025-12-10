import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Trash2, Plus, Save, X } from "lucide-react";

interface FieldSchema {
  key: string;
  label: string;
  type: "text" | "number" | "password" | "boolean" | "textarea";
  required?: boolean;
  placeholder?: string;
  description?: string;
  min?: number;
  max?: number;
  isArray?: boolean; // If true, converts comma-separated string to/from array
}

interface ObjectListFieldProps {
  value: any;
  onChange: (value: any) => void;
  schema: FieldSchema[];
  title: string;
  description?: string;
  itemName?: string; // e.g., "Server", "Indexer"
  defaultItem?: any; // Default values for new items
  renderBadges?: (item: any) => React.ReactNode; // Optional custom badges
  renderSummary?: (item: any) => string; // Optional custom summary line
}

export function ObjectListField({
  value,
  onChange,
  schema,
  title,
  description,
  itemName = "Item",
  defaultItem = {},
  renderBadges,
  renderSummary,
}: ObjectListFieldProps) {
  const [items, setItems] = useState<any[]>([]);
  const [editingItem, setEditingItem] = useState<any | null>(null);
  const [showForm, setShowForm] = useState(false);

  useEffect(() => {
    if (value) {
      try {
        const parsed = typeof value === "string" ? JSON.parse(value) : value;
        setItems(Array.isArray(parsed) ? parsed : []);
      } catch {
        setItems([]);
      }
    }
  }, [value]);

  const handleAdd = () => {
    const newItem = {
      id: crypto.randomUUID(),
      ...defaultItem,
    };
    setEditingItem(newItem);
    setShowForm(true);
  };

  const handleEdit = (item: any) => {
    setEditingItem({ ...item });
    setShowForm(true);
  };

  const handleSave = () => {
    if (!editingItem) return;

    const updatedItems = items.some((i) => i.id === editingItem.id)
      ? items.map((i) => (i.id === editingItem.id ? editingItem : i))
      : [...items, editingItem];

    setItems(updatedItems);
    onChange(updatedItems);
    setShowForm(false);
    setEditingItem(null);
  };

  const handleDelete = (id: string) => {
    const updatedItems = items.filter((i) => i.id !== id);
    setItems(updatedItems);
    onChange(updatedItems);
  };

  const handleCancel = () => {
    setShowForm(false);
    setEditingItem(null);
  };

  const handleFieldChange = (
    key: string,
    fieldValue: any,
    field?: FieldSchema,
  ) => {
    // If this is an array field, convert comma-separated string to array
    if (field?.isArray && typeof fieldValue === "string") {
      const arrayValue = fieldValue
        .split(",")
        .map((v) => v.trim())
        .filter(Boolean);
      setEditingItem({ ...editingItem, [key]: arrayValue });
    } else {
      setEditingItem({ ...editingItem, [key]: fieldValue });
    }
  };

  const renderField = (field: FieldSchema) => {
    let currentValue = editingItem?.[field.key] ?? "";

    // Convert array to comma-separated string for display
    if (field.isArray && Array.isArray(currentValue)) {
      currentValue = currentValue.join(", ");
    }

    switch (field.type) {
      case "boolean":
        return (
          <div className="flex items-center space-x-2">
            <input
              type="checkbox"
              id={field.key}
              checked={currentValue === true}
              onChange={(e) =>
                handleFieldChange(field.key, e.target.checked, field)
              }
              className="rounded"
            />
            <Label htmlFor={field.key} className="cursor-pointer">
              {field.label}
            </Label>
          </div>
        );

      case "number":
        return (
          <div className="space-y-2">
            <Label htmlFor={field.key}>
              {field.label}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </Label>
            {field.description && (
              <p className="text-xs text-muted-foreground">
                {field.description}
              </p>
            )}
            <Input
              id={field.key}
              type="number"
              value={currentValue ?? ""}
              onChange={(e) =>
                handleFieldChange(
                  field.key,
                  e.target.value ? Number(e.target.value) : null,
                  field,
                )
              }
              placeholder={field.placeholder}
              min={field.min}
              max={field.max}
            />
          </div>
        );

      case "password":
        return (
          <div className="space-y-2">
            <Label htmlFor={field.key}>
              {field.label}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </Label>
            {field.description && (
              <p className="text-xs text-muted-foreground">
                {field.description}
              </p>
            )}
            <Input
              id={field.key}
              type="password"
              value={currentValue ?? ""}
              onChange={(e) =>
                handleFieldChange(field.key, e.target.value, field)
              }
              placeholder={field.placeholder}
            />
          </div>
        );

      case "textarea":
        return (
          <div className="space-y-2">
            <Label htmlFor={field.key}>
              {field.label}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </Label>
            {field.description && (
              <p className="text-xs text-muted-foreground">
                {field.description}
              </p>
            )}
            <Textarea
              id={field.key}
              value={currentValue ?? ""}
              onChange={(e) =>
                handleFieldChange(field.key, e.target.value, field)
              }
              placeholder={field.placeholder}
              rows={3}
            />
          </div>
        );

      case "text":
      default:
        return (
          <div className="space-y-2">
            <Label htmlFor={field.key}>
              {field.label}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </Label>
            {field.description && (
              <p className="text-xs text-muted-foreground">
                {field.description}
              </p>
            )}
            <Input
              id={field.key}
              type="text"
              value={currentValue ?? ""}
              onChange={(e) =>
                handleFieldChange(field.key, e.target.value, field)
              }
              placeholder={field.placeholder}
            />
          </div>
        );
    }
  };

  const getItemTitle = (item: any) => {
    // Try to find a name field
    return (
      item.name ||
      item.title ||
      item.label ||
      `${itemName} ${item.id?.slice(0, 8)}`
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold">{title}</h3>
          {description && (
            <p className="text-sm text-muted-foreground">{description}</p>
          )}
        </div>
        {!showForm && (
          <Button onClick={handleAdd} size="sm">
            <Plus className="h-4 w-4 mr-2" />
            Add {itemName}
          </Button>
        )}
      </div>

      {/* Edit Form */}
      {showForm && editingItem && (
        <Card className="border-primary">
          <CardHeader>
            <CardTitle>
              {items.some((i) => i.id === editingItem.id)
                ? `Edit ${itemName}`
                : `New ${itemName}`}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              {schema.map((field) => (
                <div
                  key={field.key}
                  className={field.type === "textarea" ? "col-span-2" : ""}
                >
                  {renderField(field)}
                </div>
              ))}
            </div>

            <div className="flex space-x-2 pt-4">
              <Button onClick={handleSave} size="sm">
                <Save className="h-4 w-4 mr-2" />
                Save {itemName}
              </Button>
              <Button onClick={handleCancel} variant="outline" size="sm">
                <X className="h-4 w-4 mr-2" />
                Cancel
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Item List */}
      {!showForm && (
        <div className="space-y-2">
          {items.length === 0 ? (
            <Card>
              <CardContent className="pt-6">
                <p className="text-sm text-muted-foreground text-center">
                  No {title.toLowerCase()} configured. Click "Add {itemName}" to
                  get started.
                </p>
              </CardContent>
            </Card>
          ) : (
            items.map((item) => (
              <Card key={item.id}>
                <CardContent className="pt-6">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <div className="flex items-center space-x-2 mb-1">
                        <h4 className="font-medium">{getItemTitle(item)}</h4>
                        {renderBadges && renderBadges(item)}
                      </div>
                      {renderSummary ? (
                        <p className="text-sm text-muted-foreground">
                          {renderSummary(item)}
                        </p>
                      ) : null}
                    </div>
                    <div className="flex space-x-2">
                      <Button
                        onClick={() => handleEdit(item)}
                        variant="outline"
                        size="sm"
                      >
                        Edit
                      </Button>
                      <Button
                        onClick={() => handleDelete(item.id)}
                        variant="destructive"
                        size="sm"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      )}
    </div>
  );
}
