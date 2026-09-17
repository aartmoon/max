import {
  useEffect,
  useId,
  useState,
  type KeyboardEvent,
} from "react";
import { addressApi } from "../api";
import type { AddressKind, AddressSuggestion } from "../types";

type Props = {
  label: string;
  kind: AddressKind;
  value: AddressSuggestion | null;
  onChange: (value: AddressSuggestion | null) => void;
  parentObjectId?: string;
  required?: boolean;
  placeholder?: string;
};

export function AddressAutocomplete({
  label,
  kind,
  value,
  onChange,
  parentObjectId,
  required = false,
  placeholder,
}: Props) {
  const inputId = useId();
  const listId = `${inputId}-list`;
  const [query, setQuery] = useState(value?.fullAddress ?? "");
  const [items, setItems] = useState<AddressSuggestion[]>([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [activeIndex, setActiveIndex] = useState(-1);
  const canSearch = query.trim().length >= 2 || Boolean(parentObjectId);

  useEffect(() => {
    if (value) setQuery(value.fullAddress);
  }, [value]);

  useEffect(() => {
    if (value && query === value.fullAddress) {
      setItems([]);
      setMessage("");
      return;
    }
    if (!canSearch) {
      setItems([]);
      setMessage("");
      return;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(async () => {
      setLoading(true);
      setMessage("");
      try {
        const result = await addressApi.search({
          q: query.trim(),
          kind,
          parentObjectId,
          signal: controller.signal,
        });
        setItems(result.items);
        setActiveIndex(-1);
        if (result.items.length === 0) setMessage("Ничего не найдено");
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") return;
        setItems([]);
        setMessage((error as Error).message);
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }, 300);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [canSearch, kind, parentObjectId, query, value]);

  const select = (item: AddressSuggestion) => {
    setQuery(item.fullAddress);
    setItems([]);
    setMessage("");
    setActiveIndex(-1);
    onChange(item);
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (items.length === 0) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((current) => (current + 1) % items.length);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((current) => (current <= 0 ? items.length - 1 : current - 1));
    } else if (event.key === "Enter" && activeIndex >= 0) {
      event.preventDefault();
      select(items[activeIndex]);
    } else if (event.key === "Escape") {
      setItems([]);
      setActiveIndex(-1);
    }
  };

  return (
    <div className="address-autocomplete">
      <label htmlFor={inputId}>
        {label} {required && <span>*</span>}
      </label>
      <input
        id={inputId}
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={items.length > 0}
        aria-controls={listId}
        aria-activedescendant={activeIndex >= 0 ? `${listId}-${activeIndex}` : undefined}
        required={required}
        value={query}
        placeholder={placeholder}
        autoComplete="off"
        onKeyDown={onKeyDown}
        onChange={(event) => {
          const next = event.target.value;
          setQuery(next);
          if (value && next !== value.fullAddress) onChange(null);
        }}
      />
      {loading && <small className="address-status">Ищем адрес…</small>}
      {!loading && message && <small className="address-status">{message}</small>}
      {items.length > 0 && (
        <div className="address-options" id={listId} role="listbox">
          {items.map((item, index) => (
            <button
              type="button"
              role="option"
              aria-selected={index === activeIndex}
              id={`${listId}-${index}`}
              key={item.objectId}
              className={index === activeIndex ? "active" : ""}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => select(item)}
            >
              {item.fullAddress}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
