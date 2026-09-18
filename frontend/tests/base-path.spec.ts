import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.route("https://st.max.ru/js/max-web-app.js", (route) =>
    route.abort(),
  );
});

test("application and client-side navigation stay under /max", async ({
  page,
}) => {
  await page.addInitScript(() => localStorage.setItem("tvoy-dom:selected-house-object-id", "10"));
  await page.route("**/max/api/houses/resolve", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        id: "1",
        garObjectId: "10",
        address: "ул. Тестовая, д. 1",
        totalArea: 12480,
        livingArea: 9360,
        floors: 9,
        entrances: 4,
        apartments: 144,
        yearBuilt: 1987,
        organization: "УК «Тестовая»",
        manager: "Иван Иванов",
        contact: "+7 000 000-00-00",
        cadastralNumber: null,
        dataSource: "ГИС ЖКХ",
        dataUpdatedAt: "2026-09-18T12:00:00Z",
        stale: false,
        characteristics: {
          houseTypeCode: "1", houseType: "Многоквартирный", status: "APPROVED", projectSeries: null,
          condition: null, lifecycleStage: null, yearBuilt: 1987, operationYear: null, reconstructionYear: null,
          deteriorationPercent: null, deteriorationDate: null, wallMaterial: null, energyEfficiency: null,
          totalArea: 12480, livingArea: 9360, nonResidentialArea: null, residentialPremises: 144,
          residentialPremisesArea: null, residentialPremisesWithRealty: null, residentialPremisesWithRealtyArea: null,
          nonResidentialPremises: null, nonResidentialPremisesArea: null, nonResidentialPremisesNotCommon: null,
          nonResidentialPremisesNotCommonArea: null, floors: 9, entrances: 4, ownersOrShares: null,
        },
        management: {
          method: null, organizationGuid: null, shortName: "УК «Тестовая»", fullName: null,
          address: null, phone: "+7 000 000-00-00", website: null, organizationType: null,
          registryOrganizationGuid: null, inn: null, ogrn: null, chief: "Иван Иванов",
          contractStart: null, contractEnd: null,
        },
        dataSources: [
          { name: "ГИС ЖКХ · карточка дома", available: true, updatedAt: "2026-09-18T12:00:00Z", stale: false },
        ],
      }),
    }),
  );
  await page.goto("/max/");

  await expect(
    page.getByRole("heading", { name: "Хороший дом начинается с вас." }),
  ).toBeVisible();

  await page.getByRole("link", { name: "Мой дом" }).first().click();
  await expect(page).toHaveURL(/\/max\/house$/);
  await expect(
    page.getByRole("heading", { name: "Мой дом", exact: true }),
  ).toBeVisible();
  await expect(page.getByText("ул. Тестовая, д. 1")).toBeVisible();
});
