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
  characteristics: {
    houseTypeCode: "1",
    houseType: "Многоквартирный",
    status: "APPROVED",
    projectSeries: "Индивидуальный проект",
    condition: "Исправный",
    lifecycleStage: "Эксплуатация",
    yearBuilt: 1870,
    operationYear: 1870,
    reconstructionYear: 1870,
    deteriorationPercent: 66,
    deteriorationDate: "2009-12-31T00:00:00Z",
    wallMaterial: "Стены кирпичные",
    energyEfficiency: null,
    totalArea: 6720.8,
    livingArea: 2405.3,
    nonResidentialArea: 4023.6,
    residentialPremises: 15,
    residentialPremisesArea: 2405.3,
    residentialPremisesWithRealty: 15,
    residentialPremisesWithRealtyArea: 2405.3,
    nonResidentialPremises: 31,
    nonResidentialPremisesArea: 4176.4,
    nonResidentialPremisesNotCommon: 29,
    nonResidentialPremisesNotCommonArea: 4023.6,
    floors: null,
    entrances: null,
    ownersOrShares: 61,
  },
  management: {
    method: "УО",
    organizationGuid: "management-guid",
    shortName: "ГБУ «Жилищник района Арбат»",
    fullName: "Государственное бюджетное учреждение города Москвы «Жилищник района Арбат»",
    address: "121099, г. Москва, пер. Проточный, д. 9, стр. 1",
    phone: "74952305787",
    website: "https://arbatgbu.mos.ru/",
    organizationType: "L",
    registryOrganizationGuid: "registry-guid",
    inn: null,
    ogrn: "5147746267906",
    chief: null,
    contractStart: "2015-04-22T00:00:00Z",
    contractEnd: null,
  },
  dataSources: [
    { name: "ГИС ЖКХ · карточка дома", available: true, updatedAt: "2026-09-18T12:00:00Z", stale: true },
    { name: "ГИС ЖКХ · сводка помещений", available: true, updatedAt: "2026-09-18T12:00:00Z", stale: true },
  ],
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
  await expect(page.getByRole("heading", { name: "Паспорт дома" })).toBeVisible();
  await expect(page.getByText("Индивидуальный проект")).toBeVisible();
  await expect(page.getByText("15", { exact: true }).first()).toBeVisible();
  await expect(page.getByRole("heading", { name: /Жилищник района Арбат/ })).toBeVisible();
  await expect(page.getByRole("link", { name: "arbatgbu.mos.ru" })).toHaveAttribute("href", "https://arbatgbu.mos.ru/");
  await expect(page.getByText("ГИС ЖКХ · сводка помещений")).toBeVisible();
  await expect(page.getByText("Не опубликовано в ГИС ЖКХ").first()).toBeVisible();
  await expect(page.getByText(/данные могут быть устаревшими/i)).toBeVisible();
  expect(await page.evaluate(() => localStorage.getItem("tvoy-dom:selected-house-object-id"))).toBe(address.objectId);
  expect(legacyRequests).toBe(0);
  await page.getByRole("button", { name: "Выбрать другой дом" }).click();
  expect(await page.evaluate(() => localStorage.getItem("tvoy-dom:selected-house-object-id"))).toBeNull();
});

test("partial profile remains usable when the square summary is unavailable", async ({ page }) => {
  await page.unroute("**/max/api/houses/resolve");
  await page.route("**/max/api/houses/resolve", (route) => route.fulfill({
    contentType: "application/json",
    body: JSON.stringify({
      ...profile,
      characteristics: {
        ...profile.characteristics,
        yearBuilt: null,
        operationYear: 1870,
        residentialPremises: null,
        nonResidentialPremises: null,
        ownersOrShares: null,
      },
      management: { ...profile.management, website: "javascript:alert(1)" },
      dataSources: [
        profile.dataSources[0],
        { ...profile.dataSources[1], available: false },
      ],
    }),
  }));
  await page.addInitScript((objectId) => localStorage.setItem("tvoy-dom:selected-house-object-id", objectId), address.objectId);
  await page.goto("/max/house");
  await expect(page.getByRole("heading", { name: "Паспорт дома" })).toBeVisible();
  await expect(page.locator(".house-facts div").filter({ has: page.getByText("Год постройки", { exact: true }) }).locator("dd")).toHaveText("Не опубликовано в ГИС ЖКХ");
  await expect(page.locator(".house-facts div").filter({ has: page.getByText("Год ввода в эксплуатацию", { exact: true }) }).locator("dd")).toHaveText("1870");
  await expect(page.getByText("Не опубликовано в ГИС ЖКХ").first()).toBeVisible();
  await expect(page.getByText("недоступен при обновлении")).toBeVisible();
  await expect(page.getByRole("link", { name: "javascript:alert(1)" })).toHaveCount(0);
  await expect(page.getByRole("link", { name: /Создать заявку/ })).toBeVisible();
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
