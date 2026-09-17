import { expect, test } from "@playwright/test";

const address = {
  objectId: "12345",
  objectGuid: "11111111-2222-3333-4444-555555555555",
  objectKind: "house",
  displayName: "Тверская, 1",
  fullAddress: "г. Москва, ул. Тверская, д. 1",
};

const profile = {
  id: "42",
  garObjectId: address.objectId,
  objectGuid: address.objectGuid,
  address: address.fullAddress,
  cadastralNumber: null,
  totalArea: 12345.6,
  livingArea: null,
  floors: 16,
  entrances: 4,
  apartments: 120,
  yearBuilt: 1987,
  organization: "ООО УК Дом",
  manager: null,
  contact: null,
  dataSource: "ГИС ЖКХ",
  dataUpdatedAt: "2026-09-18T12:00:00Z",
  stale: true,
};

test.beforeEach(async ({ page }) => {
  await page.route("https://st.max.ru/js/max-web-app.js", (route) => route.abort());
  await page.route("**/max/api/addresses/search**", (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify({ items: [address] }) }),
  );
  await page.route("**/max/api/houses/resolve", async (route) => {
    const body = route.request().postDataJSON();
    expect(body).toEqual({ objectId: address.objectId });
    await route.fulfill({ contentType: "application/json", body: JSON.stringify(profile) });
  });
});

test("resident selects and clears a house without using the legacy endpoint", async ({ page }) => {
  let legacyRequests = 0;
  await page.route("**/max/api/house", (route) => { legacyRequests++; return route.abort(); });
  await page.goto("/max/house");
  await expect(page.getByRole("combobox", { name: "Адрес дома" })).toBeVisible();
  await page.getByRole("combobox", { name: "Адрес дома" }).fill("Тверская");
  await page.getByRole("option", { name: address.fullAddress }).click();
  await expect(page.getByRole("heading", { name: address.fullAddress })).toBeVisible();
  await expect(page.getByText("Нет данных").first()).toBeVisible();
  await expect(page.getByText(/данные могут быть устаревшими/i)).toBeVisible();
  expect(await page.evaluate(() => localStorage.getItem("tvoy-dom:selected-house-object-id"))).toBe(address.objectId);
  expect(legacyRequests).toBe(0);
  await page.getByRole("button", { name: "Выбрать другой дом" }).click();
  expect(await page.evaluate(() => localStorage.getItem("tvoy-dom:selected-house-object-id"))).toBeNull();
});

test("saved house is restored from the browser", async ({ page }) => {
  await page.addInitScript((objectId) => localStorage.setItem("tvoy-dom:selected-house-object-id", objectId), address.objectId);
  await page.goto("/max/house");
  await expect(page.getByRole("heading", { name: address.fullAddress })).toBeVisible();
});

test("resolve error is retryable", async ({ page }) => {
  await page.unroute("**/max/api/houses/resolve");
  await page.route("**/max/api/houses/resolve", (route) => route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ error: "ГИС ЖКХ временно недоступна" }) }));
  await page.addInitScript((objectId) => localStorage.setItem("tvoy-dom:selected-house-object-id", objectId), address.objectId);
  await page.goto("/max/house");
  await expect(page.getByText("ГИС ЖКХ временно недоступна")).toBeVisible();
  await expect(page.getByRole("button", { name: "Повторить" })).toBeVisible();
});
